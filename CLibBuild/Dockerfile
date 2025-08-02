FROM base:2025-07-22
RUN mkdir -p /output/include && mkdir -p /output/static-libs

ENV CAIRO=cairo-1.18.4
COPY tarballs/$CAIRO.tar.xz /tarballs/
RUN tar -xf /tarballs/$CAIRO.tar.xz && \
    cd $CAIRO && \
    meson setup build \
        --default-library=static \
        -Dc_args="-fPIC" \
        -Dcpp_args="-fPIC" \
        -Dtests=disabled \
        -Dfreetype=enabled \
        -Dfontconfig=disabled \
        -Dxlib=disabled \
        -Dxcb=disabled && \
    ninja -C build && \
    cp src/cairo.h /output/include/ && \
    cp src/cairo-version.h /output/include/ && \
    cp src/cairo-ft.h /output/include/ && \
    cp src/cairo-deprecated.h /output/include/ && \
    cp build/src/cairo-features.h /output/include/ && \
    cp build/src/libcairo.a /output/static-libs/ && \
    cd .. && rm -r $CAIRO

ENV JPEG=libjpeg-turbo-3.1.1
COPY tarballs/$JPEG.tar.gz /tarballs/
RUN tar -xf /tarballs/$JPEG.tar.gz && \
    cd $JPEG &&\
    cmake . \
        -DCMAKE_BUILD_TYPE=Release \
        -DENABLE_SHARED=OFF \
        -DENABLE_STATIC=ON \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON && \
    make -j14 && \
    cp src/turbojpeg.h /output/include/ && \
    cp libjpeg.a /output/static-libs/ && \
    cp libturbojpeg.a /output/static-libs/ && \
    cd .. && rm -r $JPEG

ENV LCMS=Little-CMS-lcms2.17
COPY tarballs/$LCMS.tar.gz /tarballs/
RUN tar -xf /tarballs/$LCMS.tar.gz && \
    cd $LCMS && \
    ./configure --disable-shared --enable-static CFLAGS="-fPIC -O2" && \
    make -j14 && \
    cp include/lcms2.h /output/include/ && \
    cp src/.libs/liblcms2.a /output/static-libs/ && \
    cd .. && rm -r $LCMS

ENV OPENJPEG=openjpeg-2.5.3
COPY tarballs/$OPENJPEG.tar.gz /tarballs/
RUN tar -xf /tarballs/$OPENJPEG.tar.gz && \
    cd $OPENJPEG && \
    cmake -G "Unix Makefiles" \
          -DCMAKE_BUILD_TYPE=Release \
          -DBUILD_SHARED_LIBS=OFF \
          -DBUILD_STATIC_LIBS=ON \
          -DBUILD_TESTING=OFF \
          -DBUILD_DOC=OFF \
          -DBUILD_CODEC=ON \
          -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
          . && \
    make -j14 && \
    cp src/lib/openjp2/openjpeg.h  /output/include/ && \
    cp src/lib/openjp2/opj_config.h  /output/include/ && \
    cp bin/libopenjp2.a /output/static-libs/ && \
    cd .. && rm -r $OPENJPEG

ENV WEBP=libwebp-1.6.0
COPY tarballs/$WEBP.tar.gz /tarballs/
RUN tar -xf /tarballs/$WEBP.tar.gz && \
    cd $WEBP && \
    ./configure CFLAGS="-fPIC" --enable-static --disable-shared && \
    make -j14 && \
    cp -r src/webp /output/include/ && \
    cp src/.libs/libwebp.a /output/static-libs/ && \
    cd .. && rm -r $WEBP

ENV FREETYPE=freetype-2.13.3
COPY tarballs/$FREETYPE.tar.xz /tarballs/
RUN tar -xf /tarballs/$FREETYPE.tar.xz && \
    cd $FREETYPE && \
    ./configure \
        CFLAGS="-fPIC" \
        --enable-static \
        --disable-shared \
        --without-harfbuzz \
        --without-brotli \
        --without-bzip2 \
        --without-png \
        --without-zlib \
        --without-graphite2 && \
    make -j14 && \
    cp -r include/* /output/include/ && \
    cp objs/.libs/libfreetype.a /output/static-libs/ && \
    cd .. && rm -r $FREETYPE

ENV PIXMAN=pixman-0.46.4
COPY tarballs/$PIXMAN.tar.xz /tarballs/
RUN tar -xf /tarballs/$PIXMAN.tar.xz && \
    cd $PIXMAN && \
    meson setup build \
        --default-library=static \
        --buildtype=release \
        -Dtests=disabled \
        -Dgtk=disabled \
        --wrap-mode=nofallback \
        -Dc_args="-fPIC" && \
    ninja -C build && \
    cp build/pixman/libpixman-1.a /output/static-libs/ && \
    cd .. && rm -r $PIXMAN

ENV POPPLER=poppler-25.07.0
COPY tarballs/$POPPLER.tar.xz /tarballs/
RUN tar -xf /tarballs/$POPPLER.tar.xz && \
    cd $POPPLER && \
    cp /output/include/*.h poppler/ && \
    sed -i 's|#include <openjpeg.h>|#include "openjpeg.h"|g' poppler/JPEG2000Stream.cc && \
    printf "set(OPENJPEG_MAJOR_VERSION \"2.5.3\")\nset(OpenJPEG_FOUND TRUE)" > cmake/modules/FindOpenJPEG.cmake && \
    printf "set(LCMS2_VERSION \"2.17\")\nset(LCMS2_FOUND TRUE)" > cmake/modules/FindLCMS2.cmake && \
    printf "set(CAIRO_FOUND TRUE)" > cmake/modules/FindCairo.cmake && \
    pacman -S --noconfirm glib2-devel && \
    cmake -G Ninja \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DENABLE_GLIB=ON \
        -DENABLE_GTK_DOC=OFF \
        -DENABLE_QT5=OFF \
        -DENABLE_QT6=OFF \
        -DENABLE_BOOST=OFF \
        -DENABLE_LIBCURL=OFF \
        -DENABLE_UNSTABLE_API_ABI_HEADERS=OFF \
        -DENABLE_UTILS=OFF \
        -DENABLE_CMS=none \
        -DENABLE_NSS3=OFF \
        -DENABLE_LIBPNG=OFF \
        -DENABLE_LIBTIFF=OFF \
        -DENABLE_LIBJPEG=ON \
        -DENABLE_ZLIB=OFF \
        -DENABLE_CAIRO=ON \
        -DBUILD_GTK_TESTS=OFF \
        -DBUILD_QT5_TESTS=OFF \
        -DBUILD_QT6_TESTS=OFF \
        -DBUILD_CPP_TESTS=OFF \
        -DBUILD_MANUAL_TESTS=OFF \
        -DENABLE_LIBOPENJPEG=openjpeg2 \
        . && \
    export STATICS="-L/$OPENJPEG/bin -l:libopenjp2.a -L/$LCMS/src/.libs -l:liblcms2.a -L/$CAIRO/build/src -l:libcairo.a" && \
    sed -i "s|-lopenjp2|${STATICS//\//\\/}|g" build.ninja && \
    ninja && \
    cp glib/poppler*.h /output/include/ && \
    cp glib/libpoppler-glib.a /output/static-libs/ && \
    cp libpoppler.a /output/static-libs/ && \
    cd .. && rm -r $POPPLER && \
    pacman -Rsc --noconfirm glib2-devel && \
    pacman -Scc --noconfirm

ENV GLIB2=glib-2.84.3
COPY tarballs/$GLIB2.tar.xz /tarballs/
RUN tar -xf /tarballs/$GLIB2.tar.xz && \
    cd $GLIB2 && \
    meson setup build \
        --default-library=static \
        --buildtype=release \
        -Dtests=disabled \
        -Db_pie=true \
        --wrap-mode=nofallback \
        -Dtests=false \
        -Dsysprof=disabled \
        -Dlibmount=disabled \
        -Dinstalled_tests=false \
        -Dc_args="-fPIC" && \
    ninja -C build && \
    cp build/glib/libglib-2.0.a /output/static-libs/ && \
    cp build/gobject/libgobject-2.0.a /output/static-libs/  && \
    cp -r glib /output/include/ && \
    cp -r build/glib/*.h /output/include/glib/ && \
    cp -r gobject /output/include/ && \
    cp -r build/gobject/*.h /output/include/gobject/ && \
    cp -r gio /output/include/ && \
    cp -r build/gio/*.h /output/include/gio/ && \
    cp -r gmodule /output/include/ && \
    cp -r build/gmodule/*.h /output/include/gmodule/ && \
    cd .. && rm -r $GLIB2

ENV FFI=libffi-3.5.1
COPY tarballs/$FFI.tar.gz /tarballs/
RUN tar -xf /tarballs/$FFI.tar.gz && \
    cd $FFI && \
    ./configure --enable-static --disable-shared CFLAGS="-fPIC -O2" && \
    make -j14 && \
    cp x86_64-pc-linux-gnu/.libs/libffi.a /output/static-libs/ && \
    cd .. && rm -r $FFI

ENV ALSA=alsa-lib-1.2.14
COPY tarballs/$ALSA.tar.bz2 /tarballs/
RUN tar -xf /tarballs/$ALSA.tar.bz2 && \
    cd $ALSA && \
    sed -i 's/handle = dlopen(filename, mode);/printf("name=%s\\npath=%s\\n", name, filename);handle = NULL;/' src/dlmisc.c && \
    sed -i 's/int st = getgrnam_r(.*);/int st=-1;/' src/pcm/pcm_direct.c && \
    sed -i '/getpwuid_r/ c\goto out;while(0){' src/userfile.c && \
    sed -i '/dlopen/ c\handle=NULL;' src/pcm/pcm_ladspa.c && \
    ./configure --enable-shared=no --enable-static=yes CFLAGS="-fPIC -O2" && \
    make -j14 && \
    cp src/.libs/libasound.a /output/static-libs/ && \
    cp include/*.h /output/include/ && \
    cd .. && rm -r $ALSA

COPY modules/pinyin/*.cpp /pinyin/
COPY modules/pinyin/make-pinyin.sh /pinyin/
COPY modules/pinyin/include/*.h /output/include/pinyin/
COPY src/stb_wrapper.h /output/include/
COPY src/stb_wrapper.c /libbar/
COPY src/plugin-pinyin.cc /pinyin/plugin.cpp
COPY ./wrapper-static.py /libbar/

RUN bash /pinyin/make-pinyin.sh && \
    python /libbar/wrapper-static.py _ /output/include/libbar_pinyin.h /pinyin/plugin.cpp

RUN cd /libbar && \
    mkdir stb && \
    git clone -v https://github.com/nothings/stb.git stb/ && \
    gcc -c stb_wrapper.c -o stb_wrapper.o -fPIC -O3 -Wall -I./ -I/output/include/ && \
    ar rcs /output/static-libs/libstb.a stb_wrapper.o

RUN sed -i '/::/d' /output/include/libbar_pinyin.h
