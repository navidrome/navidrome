#define TAGLIB_ERR_PARSE -1
#define TAGLIB_ERR_AUDIO_PROPS -2

#ifdef __cplusplus
extern "C" {
#endif

extern void go_map_put_str(unsigned long id, char *key, char *val);
extern void go_map_put_int(unsigned long id, char *key, int val);

#ifdef WIN32
int taglib_read(const wchar_t *filename, unsigned long id);
#else
int taglib_read(const char *filename, unsigned long id);
#endif

#ifdef __cplusplus
}
#endif