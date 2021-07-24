#define TAGLIB_ERR_PARSE -1
#define TAGLIB_ERR_AUDIO_PROPS -2

#ifdef __cplusplus
extern "C" {
#endif

extern void go_map_put_str(unsigned long id, char *key, char *val);
extern void go_map_put_int(unsigned long id, char *key, int val);

int taglib_read(const char *filename, unsigned long id);

#ifdef __cplusplus
}
#endif