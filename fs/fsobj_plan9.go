package fs

import (
	"bytes"
	"encoding/binary"
	"lsf/anomaly"
	"os"
	"syscall"
)

// source: http://man2.aiju.de/2/stat
//struct Dir {
//	/* system-modified data */
//	uint	type;	/* server type */
//	uint	dev;	/* server subtype */
//	/* file data */
//	Qid	qid;	/* unique id from server */
//	ulong	mode;	/* permissions */
//	ulong	atime;	/* last read time */
//	ulong	mtime;	/* last write time */
//	vlong	length;		/* file length: see <u.h> */
//	char	*name;	/* last element of path */
//	char	*uid;	/* owner name */
//	char	*gid;	/* group name */
//	char	*muid;	/* last modifier name */
//} Dir;

//struct Qid
//{
//	uvlong	path; // see: http://doc.cat-v.org/plan_9/programming/c_programming_in_plan_9
//	ulong	vers; // version? this may change ..
//	uchar	type; // OK.
//} Qid;

// determine the oid per OS specific FileInfo
// Encodes the tuple (dev,ino) as a 12 byte []byte slice.
func oid(info os.FileInfo) fsoid {
	// TODO for plan9
	// basically just make sure we're under the OIDLength limit
	panic("not implemented!")
//	if info == nil {
//		panic("BUG - info is nil")
//	}
//	var buf bytes.Buffer
//
//	fstat := info.Sys().(*syscall.Stat_t)
//	ino, dev := fstat.Ino, fstat.Dev
//
//	e := binary.Write(&buf, binary.BigEndian, dev)
//	anomaly.PanicOnError(e, "binary.Write", "device", dev)
//
//	e = binary.Write(&buf, binary.BigEndian, ino)
//	anomaly.PanicOnError(e, "binary.Write", "inode", ino)
//
//	return buf.Bytes()
}

func ignoredFiles() []string {
	return []string{"."}
}
