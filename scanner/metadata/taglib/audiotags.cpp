/***************************************************************************
    copyright            : (C) 2003 by Scott Wheeler
    email                : wheeler@kde.org
 ***************************************************************************/

/***************************************************************************
    copyright            : (C) 2014 by Nick Sellen
    email                : code@nicksellen.co.uk
 ***************************************************************************/

/***************************************************************************
 *   This library is free software; you can redistribute it and/or modify  *
 *   it  under the terms of the GNU Lesser General Public License version  *
 *   2.1 as published by the Free Software Foundation.                     *
 *                                                                         *
 *   This library is distributed in the hope that it will be useful, but   *
 *   WITHOUT ANY WARRANTY; without even the implied warranty of            *
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU     *
 *   Lesser General Public License for more details.                       *
 *                                                                         *
 *   You should have received a copy of the GNU Lesser General Public      *
 *   License along with this library; if not, write to the Free Software   *
 *   Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  *
 *   USA                                                                   *
 ***************************************************************************/

#include <stdlib.h>
#define TAGLIB_STATIC
#include <fileref.h>
#include <tfile.h>
#include <tpropertymap.h>
#include <string.h>
#include <typeinfo>

#include "audiotags.h"

static bool unicodeStrings = true;

void TagLib_free(void* pointer)
{
  free(pointer);
}

TagLib_File *audiotags_file_new(const char *filename)
{
  TagLib::File *f = TagLib::FileRef::create(filename);
  if (f == NULL || !f->isValid() || f->tag() == NULL) {
    if (f) {
      delete f;
      f = NULL;
    }
    return NULL;
  }
  return reinterpret_cast<TagLib_File *>(f);
}

void audiotags_file_close(TagLib_File *file)
{
  delete reinterpret_cast<TagLib::File *>(file);
}

void audiotags_file_properties(const TagLib_File *file, int id)
{
  const TagLib::File *f = reinterpret_cast<const TagLib::File *>(file);
  TagLib::PropertyMap tags = f->properties();
  for(TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end(); ++i) {
    for(TagLib::StringList::ConstIterator j = i->second.begin(); j != i->second.end(); ++j) {
      char *key = ::strdup(i->first.toCString(unicodeStrings));
      char *val = ::strdup((*j).toCString(unicodeStrings));
      go_map_put(id, key, val);
      free(key);
      free(val);
    }
  }
}

const TagLib_AudioProperties *audiotags_file_audioproperties(const TagLib_File *file)
{
  const TagLib::File *f = reinterpret_cast<const TagLib::File *>(file);
  return reinterpret_cast<const TagLib_AudioProperties *>(f->audioProperties());
}

const TagLib::AudioProperties *props(const TagLib_AudioProperties *audioProperties)
{
  return reinterpret_cast<const TagLib::AudioProperties *>(audioProperties); 
}

int audiotags_audioproperties_length(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->length();
}

int audiotags_audioproperties_bitrate(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->bitrate();
}

int audiotags_audioproperties_samplerate(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->sampleRate();
}

int audiotags_audioproperties_channels(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->channels();
}
