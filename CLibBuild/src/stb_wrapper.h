#ifndef STB_WRAPPER_H
#define STB_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

// Image loading
unsigned char* wrap_stbi_load(const char *filename, int *width, int *height, int *channels, int desired_channels);
unsigned char* wrap_stbi_load_from_memory(const unsigned char *buffer, int len, int *width, int *height, int *channels, int desired_channels);
void wrap_stbi_image_free(void *retval_from_stbi_load);

// Image writing
int wrap_stbi_write_png(const char *filename, int width, int height, int comp, const void *data, int stride_bytes);
int wrap_stbi_write_jpg(const char *filename, int width, int height, int comp, const void *data, int quality);

// Error handling
const char* wrap_stbi_failure_reason(void);

// Image resizing
int wrap_stbir_resize_uint8_srgb(const unsigned char *input_pixels, int input_w, int input_h, int input_stride_in_bytes,
                                unsigned char *output_pixels, int output_w, int output_h, int output_stride_in_bytes,
                                int num_channels);
int wrap_STBIR_4CHANNEL();

#ifdef __cplusplus
}
#endif

#endif // STB_WRAPPER_H
