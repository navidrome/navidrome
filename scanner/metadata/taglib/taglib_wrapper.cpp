#include <stdlib.h>
#include <string.h>
#include <typeinfo>

#define TAGLIB_STATIC
#include <aifffile.h>
#include <asffile.h>
#include <fileref.h>
#include <flacfile.h>
#include <id3v2tag.h>
#include <unsynchronizedlyricsframe.h>
#include <synchronizedlyricsframe.h>
#include <mp4file.h>
#include <mpegfile.h>
#include <opusfile.h>
#include <tpropertymap.h>
#include <vorbisfile.h>
#include <wavfile.h>

#include "taglib_wrapper.h"

// Tags necessary for M4a parsing
const char *RG_TAGS[] = {
    "replaygain_album_gain",
    "replaygain_album_peak",
    "replaygain_track_gain",
    "replaygain_track_peak"};

char *LYRICS_KEY = (char *) "lyrics";
char *CUSTOM_LYRICS = (char *) "__navidrome__lyrics";

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

  TagLib::ID3v2::Tag *id3Tags = NULL;

  // Get some extended/non-standard ID3-only tags (ex: iTunes extended frames)
  TagLib::MPEG::File *mp3File(dynamic_cast<TagLib::MPEG::File *>(f.file()));
  if (mp3File != NULL) {
    id3Tags = mp3File->ID3v2Tag();
  }

  if (id3Tags == NULL) {
    TagLib::RIFF::WAV::File *wavFile(dynamic_cast<TagLib::RIFF::WAV::File *>(f.file()));
    if (wavFile != NULL && wavFile->hasID3v2Tag()) {
      id3Tags = wavFile->ID3v2Tag();
    }
  }

  if (id3Tags == NULL) {
    TagLib::RIFF::AIFF::File *aiffFile(dynamic_cast<TagLib::RIFF::AIFF::File *>(f.file()));
    if (aiffFile && aiffFile->hasID3v2Tag()) {
      id3Tags = aiffFile->tag();
    }
  }

  // Yes, it is possible to have ID3v2 tags in FLAC. However, that can cause problems 
  // with many players, so they will not be parsed

  if (id3Tags != NULL) {
    const auto &frameListMap(id3Tags->frameListMap());

    for (const auto &kv : frameListMap) {
      if (!kv.second.isEmpty())
        if (kv.first == "USLT") {
          bool hasLyrics = false;

          for (const auto &uslt: kv.second) {
            TagLib::ID3v2::UnsynchronizedLyricsFrame *frame = dynamic_cast<TagLib::ID3v2::UnsynchronizedLyricsFrame *>(uslt);
            if (frame == NULL) continue;

            char lang[4];
            strncpy(lang, frame->language().data(), 3);
            lang[3] = '\0';
            char *val = ::strdup(frame->text().toCString(true));

            go_map_put_str(id, LYRICS_KEY, lang);
            go_map_put_str(id, LYRICS_KEY, val);
            tags.erase("LYRICS");
            hasLyrics = true;
            free(val);
          }

          if (hasLyrics) {
            go_map_put_int(id, CUSTOM_LYRICS, 1);
          }
        } else {
          tags.insert(kv.first, kv.second.front()->toString());
        }
    }
  }

  TagLib::MP4::File *m4afile(dynamic_cast<TagLib::MP4::File *>(f.file()));
  if (m4afile != NULL)
  {
    const auto itemListMap = m4afile->tag();
    {
      char buf[200];

      for (const char *key : RG_TAGS)
      {
        snprintf(buf, sizeof(buf), "----:com.apple.iTunes:%s", key);
        const auto item = itemListMap->item(buf);
        if (item.isValid())
        {
          char *dup = ::strdup(key);
          char *val = ::strdup(item.toStringList().front().toCString(true));
          go_map_put_str(id, dup, val);
          free(dup);
          free(val);
        }
      }
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
      for (const auto &value: item.second) {
        char *val = ::strdup(value.toString().toCString());
        go_map_put_str(id, key, val);
        free(val); 
      }
      free(key);
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
