#include "stb_wrapper.h"

#define STB_IMAGE_IMPLEMENTATION
#define STB_IMAGE_WRITE_IMPLEMENTATION
#define STB_IMAGE_RESIZE_IMPLEMENTATION
#include <stb/stb_image.h>
#include <stb/stb_image_write.h>
#include <stb/stb_image_resize2.h>

unsigned char* wrap_stbi_load(const char *filename, int *width, int *height, int *channels, int desired_channels) {
    return stbi_load(filename, width, height, channels, desired_channels);
}

unsigned char* wrap_stbi_load_from_memory(const unsigned char *buffer, int len, int *width, int *height, int *channels, int desired_channels) {
    return stbi_load_from_memory(buffer, len, width, height, channels, desired_channels);
}

void wrap_stbi_image_free(void *retval_from_stbi_load) {
    stbi_image_free(retval_from_stbi_load);
}

int wrap_stbi_write_png(const char *filename, int width, int height, int comp, const void *data, int stride_bytes) {
    return stbi_write_png(filename, width, height, comp, data, stride_bytes);
}

int wrap_stbi_write_jpg(const char *filename, int width, int height, int comp, const void *data, int quality) {
    return stbi_write_jpg(filename, width, height, comp, data, quality);
}

const char* wrap_stbi_failure_reason(void) {
    return stbi_failure_reason();
}

int wrap_stbir_resize_uint8_srgb(
    const unsigned char* input_pixels,
    int input_w, int input_h, int input_stride,
    unsigned char* output_pixels,
    int output_w, int output_h, int output_stride,
    int pixel_layout)
{
    unsigned char* result = stbir_resize_uint8_srgb(
        input_pixels, input_w, input_h, input_stride,
        output_pixels, output_w, output_h, output_stride,
        (stbir_pixel_layout)pixel_layout
    );
    if (result == NULL) { return 0; }
    if (result != output_pixels) {
        STBIR_FREE(result, 0);
        return 0;
    }
    return 1;
}

int wrap_STBIR_4CHANNEL() {
    return STBIR_4CHANNEL;
}
