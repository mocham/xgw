package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Constants for maximum data size
const MAX_SIZE = 50000

// log writes a formatted message to stderr
func logMsg(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// errExit logs an error message and exits with status 1
func errExit(format string, args ...interface{}) {
	logMsg(format, args...)
	os.Exit(1)
}

// getClipboard retrieves the clipboard content for the given selection and target
func getClipboard(callback func(string), selName, targetName string) error {
	// Connect to X server
	conn, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("failed to connect to X server: %v", err)
	}
	defer conn.Close()

	// Get atoms
	selAtom, err := xproto.InternAtom(conn, true, uint16(len(selName)), selName).Reply()
	if err != nil {
		return fmt.Errorf("failed to get %s atom: %v", selName, err)
	}
	targetAtom, err := xproto.InternAtom(conn, true, uint16(len(targetName)), targetName).Reply()
	if err != nil {
		return fmt.Errorf("failed to get %s atom: %v", targetName, err)
	}
	dataAtom, err := xproto.InternAtom(conn, true, uint16(len("SEL_DATA")), "SEL_DATA").Reply()
	if err != nil {
		return fmt.Errorf("failed to get SEL_DATA atom: %v", err)
	}

	// Get selection owner
	owner, err := xproto.GetSelectionOwner(conn, selAtom.Atom).Reply()
	if err != nil {
		return fmt.Errorf("failed to get selection owner: %v", err)
	}
	if owner.Owner == xproto.WindowNone {
		logMsg("No owner for selection %s", selName)
		return nil
	}

	// Create a window
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	window, err := xproto.NewWindowId(conn)
	if err != nil {
		return fmt.Errorf("failed to create window ID: %v", err)
	}
	err = xproto.CreateWindowChecked(
		conn, screen.RootDepth, window, screen.Root,
		0, 0, 10, 10, 0, xproto.WindowClassCopyFromParent,
		screen.RootVisual, xproto.CwBackPixel|xproto.CwEventMask, []uint32{0, xproto.EventMaskPropertyChange},
	).Check()
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}

	// Set window name
	progName := filepath.Base(os.Args[0])
	err = xproto.ChangePropertyChecked(
		conn, xproto.PropModeReplace, window, xproto.AtomWmName,
		xproto.AtomString, 8, uint32(len(progName)), []byte(progName),
	).Check()
	if err != nil {
		return fmt.Errorf("failed to set window name: %v", err)
	}

	// Request selection
	err = xproto.ConvertSelectionChecked(
		conn, window, selAtom.Atom, targetAtom.Atom, dataAtom.Atom, xproto.TimeCurrentTime,
	).Check()
	if err != nil {
		return fmt.Errorf("failed to convert selection: %v", err)
	}

	// Wait for SelectionNotify
	for {
		event, err := conn.WaitForEvent()
		if err != nil {
			return fmt.Errorf("failed to get event: %v", err)
		}
		if selNotify, ok := event.(xproto.SelectionNotifyEvent); ok {
			if selNotify.Requestor != window || selNotify.Selection != selAtom.Atom || selNotify.Target != targetAtom.Atom {
				return fmt.Errorf("SelectionNotify event does not match request")
			}
			if selNotify.Property == xproto.AtomNone {
				logMsg("selection lost or conversion to %s failed", targetName)
				return nil
			}
			if selNotify.Property != dataAtom.Atom {
				return fmt.Errorf("SelectionNotify property does not match request")
			}

			// Get property data
			prop, err := xproto.GetProperty(conn, false, window, dataAtom.Atom, xproto.GetPropertyTypeAny, 0, 10000).Reply()
			if err != nil {
				return fmt.Errorf("failed to get property: %v", err)
			}

			// Handle incremental data or direct output
			incrAtom, _ := xproto.InternAtom(conn, true, uint16(len("INCR")), "INCR").Reply()
			if prop.Type == incrAtom.Atom {
				logMsg("reading data incrementally: at least %d bytes", binary.BigEndian.Uint32(prop.Value[:4]))
				err = handleIncr(conn, window, dataAtom.Atom, targetName, callback)
				if err != nil {
					return err
				}
			} else {
				err = outputData(conn, prop, targetName, callback)
				if err != nil {
					return err
				}
			}

			// Delete property
			err = xproto.DeletePropertyChecked(conn, window, dataAtom.Atom).Check()
			if err != nil {
				return fmt.Errorf("failed to delete property: %v", err)
			}
			break
		}
	}
	return nil
}

// handleIncr handles incremental data transfer
func handleIncr(conn *xgb.Conn, window xproto.Window, dataAtom xproto.Atom, targetName string, callback func(string)) error {
	// Enable PropertyChangeMask
	err := xproto.ChangeWindowAttributesChecked(
		conn, window, xproto.CwEventMask, []uint32{xproto.EventMaskPropertyChange},
	).Check()
	if err != nil {
		return fmt.Errorf("failed to set event mask: %v", err)
	}

	for {
		// Delete property to request more data
		err := xproto.DeletePropertyChecked(conn, window, dataAtom).Check()
		if err != nil {
			return fmt.Errorf("failed to delete property: %v", err)
		}

		// Wait for PropertyNotify
		for {
			event, err := conn.WaitForEvent()
			if err != nil {
				return fmt.Errorf("failed to get event: %v", err)
			}
			if propNotify, ok := event.(xproto.PropertyNotifyEvent); ok && propNotify.State == xproto.PropertyNewValue && propNotify.Window == window && propNotify.Atom == dataAtom {
				break
			}
		}

		// Get property data
		prop, err := xproto.GetProperty(conn, false, window, dataAtom, xproto.GetPropertyTypeAny, 0, 10000).Reply()
		if err != nil {
			return fmt.Errorf("failed to get property: %v", err)
		}

		// End of data
		if len(prop.Value) == 0 {
			return nil
		}

		err = outputData(conn, prop, targetName, callback)
		if err != nil {
			return err
		}
	}
}

// outputData processes and outputs clipboard data
func outputData(conn *xgb.Conn, prop *xproto.GetPropertyReply, targetName string, callback func(string)) error {
	atomName, err := xproto.InternAtom(conn, true, uint16(len(targetName)), targetName).Reply()
	if err != nil {
		return fmt.Errorf("failed to get atom name: %v", err)
	}
	propTypeName, err := xproto.GetAtomName(conn, prop.Type).Reply()
	if err != nil {
		return fmt.Errorf("failed to get property type name: %v", err)
	}
	logMsg("got %s:%d, length %d", propTypeName.Name, prop.Format, len(prop.Value))

	if prop.Format == 8 {
		if prop.Type == xproto.AtomString {
			callback(string(prop.Value))
		} else if prop.Type == atomName.Atom {
			callback(string(prop.Value))
		} else {
			// Hex encode for unknown types
			var hexBuf bytes.Buffer
			for _, b := range prop.Value {
				fmt.Fprintf(&hexBuf, "%02x", b)
			}
			callback(hexBuf.String())
		}
	} else if prop.Format == 32 && prop.Type == xproto.AtomAtom {
		for i := 0; i < len(prop.Value); i += 4 {
			atom := binary.LittleEndian.Uint32(prop.Value[i : i+4])
			atomName, err := xproto.GetAtomName(conn, xproto.Atom(atom)).Reply()
			if err != nil {
				return fmt.Errorf("failed to get atom name: %v", err)
			}
			callback(fmt.Sprintf("%s\n", atomName.Name))
		}
	} else {
		for _, b := range prop.Value {
			callback(fmt.Sprintf("%d\n", b))
		}
	}
	return nil
}

// putClipboard sets the clipboard content for the given selection
func putClipboard(data, selName, targetName string) error {
	// Connect to X server
	conn, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("failed to connect to X server: %v", err)
	}
	defer conn.Close()

	// Get atoms
	selAtom, err := xproto.InternAtom(conn, true, uint16(len(selName)), selName).Reply()
	if err != nil {
		return fmt.Errorf("failed to get %s atom: %v", selName, err)
	}
	typeAtom, err := xproto.InternAtom(conn, true, uint16(len(targetName)), targetName).Reply()
	if err != nil {
		return fmt.Errorf("failed to get %s atom: %v", targetName, err)
	}
	targetsAtom, err := xproto.InternAtom(conn, true, uint16(len("TARGETS")), "TARGETS").Reply()
	if err != nil {
		return fmt.Errorf("failed to get TARGETS atom: %v", err)
	}
	timestampAtom, err := xproto.InternAtom(conn, true, uint16(len("TIMESTAMP")), "TIMESTAMP").Reply()
	if err != nil {
		return fmt.Errorf("failed to get TIMESTAMP atom: %v", err)
	}

	// Create a window
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	window, err := xproto.NewWindowId(conn)
	if err != nil {
		return fmt.Errorf("failed to create window ID: %v", err)
	}
	err = xproto.CreateWindowChecked(
		conn, screen.RootDepth, window, screen.Root,
		0, 0, 10, 10, 0, xproto.WindowClassCopyFromParent,
		screen.RootVisual, xproto.CwBackPixel|xproto.CwEventMask, []uint32{0, xproto.EventMaskPropertyChange},
	).Check()
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}

	// Set window name and get timestamp
	progName := filepath.Base(os.Args[0])
	err = xproto.ChangePropertyChecked(
		conn, xproto.PropModeReplace, window, xproto.AtomWmName,
		xproto.AtomString, 8, uint32(len(progName)), []byte(progName),
	).Check()
	if err != nil {
		return fmt.Errorf("failed to set window name: %v", err)
	}

	// Wait for PropertyNotify to get timestamp
	var selTime xproto.Timestamp
	for {
		event, err := conn.WaitForEvent()
		if err != nil {
			return fmt.Errorf("failed to get event: %v", err)
		}
		if propNotify, ok := event.(xproto.PropertyNotifyEvent); ok {
			selTime = propNotify.Time
			break
		}
	}

	// Disable event mask
	err = xproto.ChangeWindowAttributesChecked(
		conn, window, xproto.CwEventMask, []uint32{0},
	).Check()
	if err != nil {
		return fmt.Errorf("failed to disable event mask: %v", err)
	}

	// Take ownership of selection
	err = xproto.SetSelectionOwnerChecked(conn, window, selAtom.Atom, selTime).Check()
	if err != nil {
		return fmt.Errorf("failed to set selection owner: %v", err)
	}
	owner, err := xproto.GetSelectionOwner(conn, selAtom.Atom).Reply()
	if err != nil {
		return fmt.Errorf("failed to get selection owner: %v", err)
	}
	if owner.Owner != window {
		logMsg("could not take ownership of %s", selName)
		return nil
	}
	logMsg("took ownership of selection %s", selName)


	// Prepare data
	dataBytes := []byte(data)
	types := map[xproto.Atom][]byte{typeAtom.Atom: dataBytes}

	// Event loop
	for {
		event, err := conn.WaitForEvent()
		if err != nil {
			return fmt.Errorf("failed to get event: %v", err)
		}

		if selRequest, ok := event.(xproto.SelectionRequestEvent); ok && selRequest.Owner == window && selRequest.Selection == selAtom.Atom {
			client := selRequest.Requestor
			clientProp := selRequest.Property
			if clientProp == xproto.AtomNone {
				logMsg("request from obsolete client!")
				clientProp = selRequest.Target
			}

			targetName, err := xproto.GetAtomName(conn, selRequest.Target).Reply()
			if err != nil {
				return fmt.Errorf("failed to get target name: %v", err)
			}
			propName, err := xproto.GetAtomName(conn, clientProp).Reply()
			if err != nil {
				return fmt.Errorf("failed to get property name: %v", err)
			}

			logMsg("got request for %s, dest %s on 0x%08x", targetName.Name, propName.Name, client)

			var propType xproto.Atom
			var propFormat byte
			var propValue []byte

			if selRequest.Target == targetsAtom.Atom {
				propValue = make([]byte, 8)
				binary.LittleEndian.PutUint32(propValue[0:4], uint32(targetsAtom.Atom))
				binary.LittleEndian.PutUint32(propValue[4:8], uint32(typeAtom.Atom))
				propType = xproto.AtomAtom
				propFormat = 32
			} else if selRequest.Target == timestampAtom.Atom {
				propValue = make([]byte, 4)
				binary.LittleEndian.PutUint32(propValue, uint32(selTime))
				propType = xproto.AtomInteger
				propFormat = 32
			} else if data, ok := types[selRequest.Target]; ok {
				propValue = data
				propType = selRequest.Target
				propFormat = 8
			} else {
				logMsg("refusing conversion to %s", targetName.Name)
				clientProp = xproto.AtomNone
			}

			if clientProp != xproto.AtomNone {
				err = xproto.ChangePropertyChecked(
					conn, xproto.PropModeReplace, client, clientProp,
					propType, propFormat, uint32(len(propValue)/int(propFormat/8)), propValue,
				).Check()
				if err != nil {
					return fmt.Errorf("failed to set property: %v", err)
				}
			}

			// Send SelectionNotify
			selNotify := xproto.SelectionNotifyEvent{
				Time:      selRequest.Time,
				Requestor: selRequest.Requestor,
				Selection: selRequest.Selection,
				Target:    selRequest.Target,
				Property:  clientProp,
			}
			err = xproto.SendEventChecked(
				conn, false, client, xproto.EventMaskNoEvent, string(selNotify.Bytes()),
			).Check()
			if err != nil {
				return fmt.Errorf("failed to send SelectionNotify: %v", err)
			}
		} 
	}
}

// xselMain mimics xsel behavior
func xselMain(args []string) error {
	selName := "CLIPBOARD"
	action := "get"

	for _, arg := range args[1:] {
		if strings.Contains(arg, "p") {
			selName = "PRIMARY"
		}
		if strings.Contains(arg, "s") {
			selName = "SECONDARY"
		}
		if strings.Contains(arg, "b") {
			selName = "CLIPBOARD"
		}
		if strings.Contains(arg, "o") {
			action = "get"
		}
		if strings.Contains(arg, "i") {
			action = "put"
		}
		if strings.Contains(arg, "d") {
			action = "put_debug"
		}
	}

	if action == "get" {
		return getClipboard(func(s string) { fmt.Print(s) }, selName, "UTF8_STRING")
	} else if action == "put" || action == "put_debug" {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, MAX_SIZE))
		if err != nil {
			return fmt.Errorf("failed to read stdin: %v", err)
		}
		return putClipboard(string(data), selName, "UTF8_STRING")
	}
	return fmt.Errorf("unknown action: %s", action)
}

func main() {
	if err := xselMain(os.Args); err != nil {
		errExit("Error: %v", err)
	}
}
