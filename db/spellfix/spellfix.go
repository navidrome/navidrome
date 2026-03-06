//go:build sqlite_spellfix

package spellfix

/*
#cgo CFLAGS: -I${SRCDIR} -Wno-deprecated-declarations

// Avoid duplicate symbol conflicts with go-sqlite3.
// Rename the api pointer and entry point to unique names for this compilation unit.
#define sqlite3_api sqlite3_api_spellfix
#define sqlite3_spellfix_init sqlite3_spellfix_init_local

// Compile the extension into this binary.
// spellfix.c includes sqlite3ext.h and declares SQLITE_EXTENSION_INIT1.
#include "spellfix.c"

// sqlite3ext.h redefines sqlite3_auto_extension as a macro through the api
// struct. Undo that so we can call the real C function directly.
#undef sqlite3_auto_extension

// Provided by the SQLite library linked via go-sqlite3.
extern int sqlite3_auto_extension(void(*)(void));

// Register spellfix so it is available on every new sqlite3_open() connection.
static void register_spellfix(void) {
	sqlite3_auto_extension((void(*)(void))sqlite3_spellfix_init_local);
}
*/
import "C"

func init() {
	C.register_spellfix()
}
