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
#include <fstream>
#include <regex>

#include "taglib_wrapper.h"

char has_cover(const TagLib::FileRef f);

static char TAGLIB_VERSION[16];

// Parses an .lrc file and extracts the lyrics with timestamps
bool parse_lrc(const std::string& lrcFileName, unsigned long id) {
    std::ifstream lrcFile(lrcFileName);
    if (!lrcFile.is_open()) {
        return false; // .lrc file not found
    }

    std::string line;
    std::regex timeRegex(R"(\[([0-9]{2}):([0-9]{2})(?:\.([0-9]{2,3}))?\](.*))");
    std::smatch match;
    bool hasLyrics = false;

    while (std::getline(lrcFile, line)) {
        if (std::regex_match(line, match, timeRegex)) {
            int minutes = std::stoi(match[1].str());
            int seconds = std::stoi(match[2].str());
            int milliseconds = match[3].matched ? std::stoi(match[3].str()) : 0;

            int timeInMs = (minutes * 60 * 1000) + (seconds * 1000) + milliseconds;
            std::string lyricsText = match[4].str();

            // Add each lyric line to the Go map
            if (!lyricsText.empty()) {
                go_map_put_lyric_line(id, "lrc", (char*)lyricsText.c_str(), timeInMs);
                hasLyrics = true;
            }
        }
    }

    lrcFile.close();
    return hasLyrics;
}

char* taglib_version() {
    snprintf((char *)TAGLIB_VERSION, 16, "%d.%d.%d", TAGLIB_MAJOR_VERSION, TAGLIB_MINOR_VERSION, TAGLIB_PATCH_VERSION);
    return (char *)TAGLIB_VERSION;
}

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
  go_map_put_int(id, (char *)"duration", props->lengthInSeconds());
  go_map_put_int(id, (char *)"lengthinmilliseconds", props->lengthInMilliseconds());
  go_map_put_int(id, (char *)"bitrate", props->bitrate());
  go_map_put_int(id, (char *)"channels", props->channels());
  go_map_put_int(id, (char *)"samplerate", props->sampleRate());

  // Create a map to collect all the tags
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
    const auto &frames = id3Tags->frameListMap();

    for (const auto &kv: frames) {
      if (kv.first == "USLT") {
        for (const auto &tag: kv.second) {
          TagLib::ID3v2::UnsynchronizedLyricsFrame *frame = dynamic_cast<TagLib::ID3v2::UnsynchronizedLyricsFrame *>(tag);
          if (frame == NULL) continue;

          tags.erase("LYRICS");

          const auto bv = frame->language();
          char language[4] = {'x', 'x', 'x', '\0'};
          if (bv.size() == 3) {
            strncpy(language, bv.data(), 3);
          }

          char *val = (char *)frame->text().toCString(true);

          go_map_put_lyrics(id, language, val);
        }
      } else if (kv.first == "SYLT") {
        for (const auto &tag: kv.second) {
          TagLib::ID3v2::SynchronizedLyricsFrame *frame = dynamic_cast<TagLib::ID3v2::SynchronizedLyricsFrame *>(tag);
          if (frame == NULL) continue;

          const auto bv = frame->language();
          char language[4] = {'x', 'x', 'x', '\0'};
          if (bv.size() == 3) {
            strncpy(language, bv.data(), 3);
          }

          const auto format = frame->timestampFormat();
          if (format == TagLib::ID3v2::SynchronizedLyricsFrame::AbsoluteMilliseconds) {

            for (const auto &line: frame->synchedText()) {
              char *text = (char *)line.text.toCString(true);
              go_map_put_lyric_line(id, language, text, line.time);
            }
          } else if (format == TagLib::ID3v2::SynchronizedLyricsFrame::AbsoluteMpegFrames) {
            const int sampleRate = props->sampleRate();

            if (sampleRate != 0) {
              for (const auto &line: frame->synchedText()) {
                const int timeInMs = (line.time * 1000) / sampleRate;
                char *text = (char *)line.text.toCString(true);
                go_map_put_lyric_line(id, language, text, timeInMs);
              }
            }
          }
        }
      } else {
        if (!kv.second.isEmpty()) {
          tags.insert(kv.first, kv.second.front()->toString());
        }
      }
    }
  }

  // M4A may have some iTunes specific tags
  TagLib::MP4::File *m4afile(dynamic_cast<TagLib::MP4::File *>(f.file()));
  if (m4afile != NULL) {
    const auto itemListMap = m4afile->tag()->itemMap();
    for (const auto item: itemListMap) {
      char *key = (char *)item.first.toCString(true);
      for (const auto value: item.second.toStringList()) {
        char *val = (char *)value.toCString(true);
        go_map_put_m4a_str(id, key, val);
      }
    }
  }

  // WMA/ASF files may have additional tags not captured by the general iterator
  TagLib::ASF::File *asfFile(dynamic_cast<TagLib::ASF::File *>(f.file()));
  if (asfFile != NULL) {
    const TagLib::ASF::Tag *asfTags{asfFile->tag()};
    const auto itemListMap = asfTags->attributeListMap();
    for (const auto item : itemListMap) {
      tags.insert(item.first, item.second.front().toString());
    }
  }

  // Send all collected tags to the Go map
  for (TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end();
       ++i) {
    char *key = (char *)i->first.toCString(true);
    for (TagLib::StringList::ConstIterator j = i->second.begin();
         j != i->second.end(); ++j) {
      char *val = (char *)(*j).toCString(true);
      go_map_put_str(id, key, val);
    }
  }

  
  // Attempt to load lyrics from a matching .lrc file
  std::string audioFileName(filename);
  std::string lrcFileName = audioFileName + ".lrc"; // Assuming .lrc has the same base name
  bool hasLrcLyrics = parse_lrc(lrcFileName, id);

  if (hasLrcLyrics) {
      go_map_put_str(id, (char *)"has_lrc_lyrics", (char *)"true");
  }

  return 0;
}

// Detect if the file has cover art. Returns 1 if the file has cover art, 0 otherwise.
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
