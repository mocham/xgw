#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <pinyin/pinyinime.h>

static int initialized = 0;

// Helper function to convert UTF-16 (char16) to UTF-8
// Returns a dynamically allocated string; caller must free it
size_t utf16_to_utf8(const ime_pinyin::char16* utf16, size_t len, char *output, size_t pos, size_t cardinality) {
    if (!output) return pos;
    size_t j = pos;
    if (j >= cardinality) return pos;
    for (size_t i = 0; i < len && utf16[i]; i++) {
        uint16_t c = utf16[i];
        if (c <= 0x7F) {
            output[j++] = (char)c;
            if (j >= cardinality) return j;
        } else if (c <= 0x7FF) {
            output[j++] = (char)(0xC0 | (c >> 6));
            if (j >= cardinality) return j;
            output[j++] = (char)(0x80 | (c & 0x3F));
            if (j >= cardinality) return j;
        } else {
            output[j++] = (char)(0xE0 | (c >> 12));
            if (j >= cardinality) return j;
            output[j++] = (char)(0x80 | ((c >> 6) & 0x3F));
            if (j >= cardinality) return j;
            output[j++] = (char)(0x80 | (c & 0x3F));
            if (j >= cardinality) return j;
        }
    }
    output[j++] = '\n';
    return j;
}

// Helper function to parse input: replace first '+' with '\0' and return '+' count
// Modifies input string in-place; returns number of '+' symbols
static size_t strip_suf(char* input, size_t len) {
    if (len <= 0) { return 0; }
    for (len-- ; len >= 0; len--) {
        if (input[len] == '+') {
            input[len] = '\0'; 
        } else {
            break;
        }
    }
    return len + 1;
}

extern "C" {

int init_pinyin(const char* sys_dict_path, const char* usr_dict_path) {
    if (!ime_pinyin::im_open_decoder(sys_dict_path, usr_dict_path)) {
        return -1;
    }
    initialized = 1;
    printf("libgooglepinyin initialized");
    return 0;
}

}
extern "C" {

void cleanup_pinyin() {
    if (initialized == 1) {
        printf("libgooglepinyin cleanup");
        ime_pinyin::im_close_decoder();
    }
}
}
extern "C" {

void get_pinyin_candidates(char* input_str, char* output_str, size_t output_cardinality) {
    // Initialize the decoder
    if (output_cardinality <= 0 || !output_str) { return; }
    if (!input_str) { 
        output_str[0] = '\0';
        return;
    }

    // Parse input to get Pinyin and count of '+' symbols
    size_t input_len = strlen(input_str);
    size_t actual_len = strip_suf(input_str, input_len);
    if (actual_len <= 0) {
        output_str[0] = '\0';
        return;
    }

    // Perform search with the input Pinyin string
    size_t cand_num = ime_pinyin::im_search(input_str, actual_len);
    if (cand_num == 0) {
        output_str[0] = '\0';
        return;
    }

    // Print up to 5 candidates
    ime_pinyin::char16 cand_buf[256];
    size_t first_cand = (input_len - actual_len) * 5;
    size_t last_cand = first_cand + 5;
    if (last_cand > cand_num) { last_cand = cand_num; }
    size_t pos = 0;
    for (size_t i = first_cand; i < last_cand; i++) {
        if (ime_pinyin::im_get_candidate(i, cand_buf, 256)) {
            pos = utf16_to_utf8(cand_buf, 256, output_str, pos, output_cardinality - 1);
        }
    }
    if (pos >= output_cardinality) { pos = output_cardinality - 1; }
    output_str[pos] = '\0';
    return;
}

}

/* int main(int argc, char* argv[]) {
    char* output = (char*)malloc(1024);
    get_pinyin_candidates("/home/lz/Bar/User_Data/dict_pinyin.dat", "/home/lz/Bar/User_Data/userdict_pinyin.dat", argv[1], output, 1024);
    printf("%s", output);
    free(output);
    return 0;
}*/
