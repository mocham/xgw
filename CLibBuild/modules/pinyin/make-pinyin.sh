#! /usr/bin/bash
cd /pinyin
declare -a OBJECTS
mkdir -p cache
for src in $(ls *.cpp); do
    obj="cache/${src%.*}.o"
    echo "$src -> $obj"
    g++ $CXX -fPIC -O3 -c "$src" -I/output/include -o "$obj" || exit 1
    OBJECTS+=("$obj")
done
ar rcs /output/static-libs/libgooglepinyin.a "${OBJECTS[@]}"
