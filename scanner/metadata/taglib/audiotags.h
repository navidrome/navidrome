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

#ifdef __cplusplus
extern "C" {
#endif

typedef struct { int dummy; } TagLib_File;
typedef struct { int dummy; } TagLib_AudioProperties;

extern void go_map_put(int id, char *key, char *val);

void audiotags_free(void* pointer);
TagLib_File *audiotags_file_new(const char *filename);
void audiotags_file_close(TagLib_File *file);
void audiotags_file_properties(const TagLib_File *file, int id);
const TagLib_AudioProperties *audiotags_file_audioproperties(const TagLib_File *file);

int audiotags_audioproperties_length(const TagLib_AudioProperties *audioProperties);
int audiotags_audioproperties_bitrate(const TagLib_AudioProperties *audioProperties);
int audiotags_audioproperties_samplerate(const TagLib_AudioProperties *audioProperties);
int audiotags_audioproperties_channels(const TagLib_AudioProperties *audioProperties);

#ifdef __cplusplus
}
#endif