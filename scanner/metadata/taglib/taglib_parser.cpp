#include <stdlib.h>
#include <string.h>
#include <typeinfo>

#define TAGLIB_STATIC
#include <fileref.h>
#include <id3v2tag.h>
#include <mpegfile.h>
#include <tpropertymap.h>

#include "taglib_parser.h"

int taglib_read(const char *filename, unsigned long id) {
  TagLib::FileRef f(filename, true, TagLib::AudioProperties::Fast);

  if (f.isNull()) {
    return TAGLIB_ERR_PARSE;
  }

  if (!f.audioProperties()) {
    return TAGLIB_ERR_AUDIO_PROPS;
  }

  const TagLib::AudioProperties *props(f.audioProperties());
  go_map_put_int(id, (char *)"length", props->length());
  go_map_put_int(id, (char *)"bitrate", props->bitrate());

  TagLib::PropertyMap tags = f.file()->properties();

  TagLib::MPEG::File *mp3File(dynamic_cast<TagLib::MPEG::File *>(f.file()));
  if (mp3File != NULL) {
    if (mp3File->ID3v2Tag()) {
      const auto &frameListMap(mp3File->ID3v2Tag()->frameListMap());

      if (!frameListMap["TCMP"].isEmpty())
        tags.insert("compilation", frameListMap["TCMP"].front()->toString());
      if (!frameListMap["TSST"].isEmpty())
        tags.insert("discsubtitle", frameListMap["TSST"].front()->toString());
    }
  }

  for (TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end();
       ++i) {
    for (TagLib::StringList::ConstIterator j = i->second.begin();
         j != i->second.end(); ++j) {
      char *key = ::strdup(i->first.toCString(true));
      char *val = ::strdup((*j).toCString(true));
      go_map_put_str(id, key, val);
      free(key);
      free(val);
    }
  }

  return 0;
}
