This is a copy of src/image/jpeg from go-1.16, with tweakers to
reader.go to be more permissive of "short Huffman data" errors
This also required making a copy of the "internals" code

BEWARE - image/RegisterFormat() _appends_ the registered entries, so only
the first-registered routine for a given format is called; there seems to be no
effective way to register a format override after initialization
