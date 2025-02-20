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

extern void goPutM4AStr(unsigned long id, char *key, char *val);
extern void goPutStr(unsigned long id, char *key, char *val);
extern void goPutInt(unsigned long id, char *key, int val);
extern void goPutLyrics(unsigned long id, char *lang, char *val);
extern void goPutLyricLine(unsigned long id, char *lang, char *text, int time);
int taglib_read(const FILENAME_CHAR_T *filename, unsigned long id);
char* taglib_version();

#ifdef __cplusplus
}
#endif
