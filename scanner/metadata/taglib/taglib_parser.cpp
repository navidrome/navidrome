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

  // Add audio properties to the tags
  const TagLib::AudioProperties *props(f.audioProperties());
  go_map_put_int(id, (char *)"length", props->length());
  go_map_put_int(id, (char *)"bitrate", props->bitrate());

  TagLib::PropertyMap tags = f.file()->properties();

  // Make sure at least the basic properties are extracted
  TagLib::Tag *basic = f.file()->tag();
  if (!basic->isEmpty()) {
    if (!basic->title().isEmpty()) {
      tags.insert("_title", basic->title());
    }
    if (!basic->artist().isEmpty()) {
      tags.insert("_artist", basic->artist());
    }
    if (!basic->album().isEmpty()) {
      tags.insert("_album", basic->album());
    }
    if (!basic->genre().isEmpty()) {
      tags.insert("_genre", basic->genre());
    }
    if (basic->year() > 0) {
      tags.insert("_year", TagLib::String::number(basic->year()));
    }
    if (basic->track() > 0) {
      tags.insert("_track", TagLib::String::number(basic->track()));
    }
  }

  // Get some extended/non-standard ID3-only tags (ex: iTunes extended frames)
  TagLib::MPEG::File *mp3File(dynamic_cast<TagLib::MPEG::File *>(f.file()));
  if (mp3File != NULL) {
    if (mp3File->ID3v2Tag()) {
      const auto &frameListMap(mp3File->ID3v2Tag()->frameListMap());

      for (const auto& kv : frameListMap) {
        if (!kv.second.isEmpty())
          tags.insert(kv.first, kv.second.front()->toString());
      }
    }
  }

  // Get only the first occurrence of each tag (for now)
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
