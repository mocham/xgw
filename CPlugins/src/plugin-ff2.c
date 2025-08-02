#include <ft2build.h>
#include FT_FREETYPE_H
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

uint32_t utf8_to_codepoint(const char* utf8) {
    unsigned char c = *(const unsigned char*)utf8;
    if (c < 0x80) return c;
    if (c < 0xE0) return ((c & 0x1F) << 6)  | (utf8[1] & 0x3F);
    if (c < 0xF0) return ((c & 0x0F) << 12) | ((utf8[1] & 0x3F) << 6) | (utf8[2] & 0x3F);
    return ((c & 0x07) << 18) | ((utf8[1] & 0x3F) << 12) | ((utf8[2] & 0x3F) << 6) | (utf8[3] & 0x3F);
}

static FT_Library ft_lib = NULL;
static FT_Face fonts[] = {
    NULL, NULL, NULL, NULL
};

int ft_init(char* font0, char* font1, char* font2, int height) {
    if (FT_Init_FreeType(&ft_lib)) return 0;
    if (FT_New_Face(ft_lib, font0, 0, &fonts[0]) == 0) {
        FT_Set_Pixel_Sizes(fonts[0], 0, height);
    }
    if (FT_New_Face(ft_lib, font1, 0, &fonts[1]) == 0) {
        FT_Set_Pixel_Sizes(fonts[1], 0, height);
    }
    if (FT_New_Face(ft_lib, font2, 0, &fonts[2]) == 0) {
        FT_Set_Pixel_Sizes(fonts[2], 0, height);
    }
    return 1;
}

void ft_cleanup() {
    for (int i = 0; fonts[i]; i++) { FT_Done_Face(fonts[i]); }
    FT_Done_FreeType(ft_lib);
}

int render_char_to_rgba(
    const char* utf8_char,
    uint32_t bg,
    uint32_t fg,
    uint32_t** out_buffer,
    int* out_width,
    int* out_height,
    int* out_baseline
) {
    int error, i;
    FT_Face face = NULL;
    uint32_t codepoint = utf8_to_codepoint(utf8_char);
    FT_UInt glyph_index;
    for (i = 0; fonts[i]; i++) {
        glyph_index = FT_Get_Char_Index(fonts[i], codepoint);
        if (glyph_index) {
            face = fonts[i];
            break;
        }
    }
    if (glyph_index == 0 || face == NULL || FT_Load_Glyph(face, glyph_index, FT_LOAD_DEFAULT) != 0) {
        return -1;
    }
    // Render to 8-bit grayscale bitmap
    if ( FT_Render_Glyph(face->glyph, FT_RENDER_MODE_NORMAL) != 0) {
        return -1;
    }
    FT_Bitmap* bitmap = &face->glyph->bitmap;
    // Allocate output buffer
    *out_width = bitmap->width;
    *out_height = bitmap->rows;
    *out_baseline = face->glyph->bitmap_top;
    *out_buffer = (uint32_t*)malloc(*out_width * *out_height * sizeof(uint32_t));
    if (!*out_buffer) {
        return -1;
    }
    // Render glyph with alpha blending
    for (int y = 0; y < bitmap->rows; y++) {
        for (int x = 0; x < bitmap->width; x++) {
            uint8_t alpha = bitmap->buffer[y * bitmap->pitch + x];
            uint32_t* pixel = &(*out_buffer)[y * *out_width + x];
            if (alpha > 0) {
                if (alpha == 255) {
                    *pixel = fg;
                } else {
                    // Blend colors (approximate fast version)
                    uint32_t bg_rb = bg& 0x00FF00FF;
                    uint32_t bg_g = bg& 0x0000FF00;
                    uint32_t fg_rb = fg& 0x00FF00FF;
                    uint32_t fg_g = fg& 0x0000FF00;
                    uint32_t rb = ((fg_rb - bg_rb) * alpha >> 8) + bg_rb;
                    uint32_t g = ((fg_g - bg_g) * alpha >> 8) + bg_g;
                    *pixel = (rb & 0x00FF00FF) | (g & 0x0000FF00) | (0xFF000000);
                }
            } else { *pixel = bg; }
        }
    }
    return 0;
}

int post_process_glyph(
    uint32_t* src,
    uint32_t* dst,
    int src_width,
    int src_height,
    int y_off,
    int width,  // Returns new width
    int height,       // Fixed output height
    uint32_t bg
) {
    if (3*src_width < 2*width) { width = width/2; }
    int x_off = (width - src_width) / 2;
    int i, y;
    for (i=0; i<width; i++) {
        dst[i] = bg;
    }
    if (y_off < 0) {
        y_off = 0;
    } else {
        for (y=1; y < y_off; y++){
            memcpy(dst+width*y, dst, width*sizeof(uint32_t));
        }
    }
    for (y = src_height+y_off; y < height; y++) {
        memcpy(dst+width*y, dst, width*sizeof(uint32_t));
    }
    for (y = 0; y < src_height; y++) {
        if (x_off > 0) {
            for (i=0; i<x_off; i++) {
                dst[(y + y_off) * width + i] = bg;
            }
            memcpy(
                &dst[(y + y_off) * width + x_off],
                &src[y * src_width],
                src_width * sizeof(uint32_t)
            );
        } else {
            memcpy(
                &dst[(y + y_off) * width],
                &src[y * src_width-x_off],
                src_width * sizeof(uint32_t)
            );
        }
        for (int i=src_width+x_off; i<width; i++) {
            dst[(y + y_off) * width + i] = bg;
        }
    }
    return width;
}
int make_ff2_glyph(
    char* utf8_char,    // Null-terminated UTF-8 character, e.g. "A\0"
    uint32_t fg_color,    // 0xAARRGGBB
    uint32_t bg_color,    // 0xAARRGGBB
    int out_width,
    int out_height,
    int out_baseline,
    uint32_t* dst      // Pre-allocated output buffer
) {
    uint32_t* buffer;
    int width, height, baseline;
    int result = render_char_to_rgba(
        utf8_char,
        bg_color,
        fg_color,
        &buffer,
        &width,
        &height,
        &baseline
    );
    if (result != 0) return 0;
    int ow = post_process_glyph(buffer, dst, width, height, out_baseline - baseline, out_width, out_height, bg_color);
    //printf("Glyph [%s] %d (%d,%d) -> %d\n", utf8_char, baseline, width, height, ow);
    free(buffer);
    return ow;
}
