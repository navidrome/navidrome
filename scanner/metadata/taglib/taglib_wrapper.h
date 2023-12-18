#define TAGLIB_ERR_PARSE -1
#define TAGLIB_ERR_AUDIO_PROPS -2

#ifdef __cplusplus
extern "C" {
#endif

#ifdef WIN32
#define FILENAME_CHAR_T wchar_t
#else
#define FILENAME_CHAR_T char
#endif

extern void go_map_put_m4a_str(unsigned long id, char *key, char *val);
extern void go_map_put_str(unsigned long id, char *key, char *val);
extern void go_map_put_int(unsigned long id, char *key, int val);
extern void go_map_put_lyrics(unsigned long id, char *lang, char *val);
extern void go_map_put_lyric_line(unsigned long id, char *lang, char *text, int time);
int taglib_read(const FILENAME_CHAR_T *filename, unsigned long id);

#ifdef __cplusplus
}
#endif
