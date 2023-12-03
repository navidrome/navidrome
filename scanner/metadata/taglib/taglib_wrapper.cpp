#include <stdlib.h>
#include <string.h>
#include <typeinfo>

#define TAGLIB_STATIC
#include <asffile.h>
#include <fileref.h>
#include <flacfile.h>
#include <id3v2tag.h>
#include <mp4file.h>
#include <mpegfile.h>
#include <opusfile.h>
#include <tpropertymap.h>
#include <vorbisfile.h>

#include "taglib_wrapper.h"

const char *ITUNES_KEY = "----:com.apple.iTunes:";
const size_t ITUNES_LEN = strlen(ITUNES_KEY);

char has_cover(const TagLib::FileRef f);

int taglib_read(const FILENAME_CHAR_T *filename, unsigned long id) {
  TagLib::FileRef f(filename, true, TagLib::AudioProperties::Fast);

  if (f.isNull()) {
    return TAGLIB_ERR_PARSE;
  }

  if (!f.audioProperties()) {
    return TAGLIB_ERR_AUDIO_PROPS;
  }

  // Add audio properties to the tags
  const TagLib::AudioProperties *props(f.audioProperties());
  go_map_put_int(id, (char *)"duration", props->length());
  go_map_put_int(id, (char *)"lengthinmilliseconds", props->lengthInMilliseconds());
  go_map_put_int(id, (char *)"bitrate", props->bitrate());
  go_map_put_int(id, (char *)"channels", props->channels());

  TagLib::PropertyMap tags = f.file()->properties();

  // Make sure at least the basic properties are extracted
  TagLib::Tag *basic = f.file()->tag();
  if (!basic->isEmpty()) {
    if (!basic->title().isEmpty()) {
      tags.insert("title", basic->title());
    }
    if (!basic->artist().isEmpty()) {
      tags.insert("artist", basic->artist());
    }
    if (!basic->album().isEmpty()) {
      tags.insert("album", basic->album());
    }
    if (basic->year() > 0) {
      tags.insert("date", TagLib::String::number(basic->year()));
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

      for (const auto &kv : frameListMap) {
        if (!kv.second.isEmpty())
          tags.insert(kv.first, kv.second.front()->toString());
      }
    }
  }

  TagLib::MP4::File *m4afile(dynamic_cast<TagLib::MP4::File *>(f.file()));
  if (m4afile != NULL)
  {
    const auto tag = m4afile->tag();
    const auto itemListMap = tag->itemMap();
    for (const auto item: itemListMap) {
      char *key = ::strdup(item.first.toCString(true));
      // the strlen check is to deal with cases where there is a tag that is just
      // ----:com.apple.iTunes:\x00. Note that the check at ITUNES_LEN is safe 
      // because key (via toCString) is a C-string (null-terminated)
      if (strncmp(key, ITUNES_KEY, ITUNES_LEN) == 0 && key[ITUNES_LEN] != '\0') {
        char *val = ::strdup(item.second.toStringList().front().toCString(true));
        go_map_put_str(id, (key + ITUNES_LEN), val);
        free(val);
      } 

      free(key);
    }
  }

  // WMA/ASF files may have additional tags not captured by the general iterator
  TagLib::ASF::File *asfFile(dynamic_cast<TagLib::ASF::File *>(f.file()));
  if (asfFile != NULL) 
  {
    const TagLib::ASF::Tag *asfTags{asfFile->tag()};
    const auto itemListMap = asfTags->attributeListMap();
    for (const auto item : itemListMap) {
      char *key = ::strdup(item.first.toCString(true));
      char *val = ::strdup(item.second.front().toString().toCString());
      go_map_put_str(id, key, val);
      free(key);
      free(val); 
    }

    // Compilation tag needs to be handled differently
    const auto compilation = asfTags->attribute("WM/IsCompilation");
    if (!compilation.isEmpty()) {
      char *val = ::strdup(compilation.front().toString().toCString());
      go_map_put_str(id, (char *)"compilation", val);
      free(val);
    }
  }

  if (has_cover(f)) {
    go_map_put_str(id, (char *)"has_picture", (char *)"true");
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

char has_cover(const TagLib::FileRef f) {
  char hasCover = 0;
  // ----- MP3
  if (TagLib::MPEG::File *
      mp3File{dynamic_cast<TagLib::MPEG::File *>(f.file())}) {
    if (mp3File->ID3v2Tag()) {
      const auto &frameListMap{mp3File->ID3v2Tag()->frameListMap()};
      hasCover = !frameListMap["APIC"].isEmpty();
    }
  }
  // ----- FLAC
  else if (TagLib::FLAC::File *
           flacFile{dynamic_cast<TagLib::FLAC::File *>(f.file())}) {
    hasCover = !flacFile->pictureList().isEmpty();
  }
  // ----- MP4
  else if (TagLib::MP4::File *
           mp4File{dynamic_cast<TagLib::MP4::File *>(f.file())}) {
    auto &coverItem{mp4File->tag()->itemMap()["covr"]};
    TagLib::MP4::CoverArtList coverArtList{coverItem.toCoverArtList()};
    hasCover = !coverArtList.isEmpty();
  }
  // ----- Ogg
  else if (TagLib::Ogg::Vorbis::File *
           vorbisFile{dynamic_cast<TagLib::Ogg::Vorbis::File *>(f.file())}) {
    hasCover = !vorbisFile->tag()->pictureList().isEmpty();
  }
  // ----- Opus
  else if (TagLib::Ogg::Opus::File *
           opusFile{dynamic_cast<TagLib::Ogg::Opus::File *>(f.file())}) {
    hasCover = !opusFile->tag()->pictureList().isEmpty();
  }
  // ----- WMA
  if (TagLib::ASF::File *
      asfFile{dynamic_cast<TagLib::ASF::File *>(f.file())}) {
    const TagLib::ASF::Tag *tag{asfFile->tag()};
    hasCover = tag && tag->attributeListMap().contains("WM/Picture");
  }

  return hasCover;
}
