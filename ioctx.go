package rados

// #cgo LDFLAGS: -lrados
// #include <stdlib.h>
// #include <rados/librados.h>
import "C"

import "unsafe"

// PoolStat represents Ceph pool statistics.
type PoolStat struct {
  // space used in bytes
  Num_bytes uint64
  // space used in KB
  Num_kb uint64
  // number of objects in the pool
  Num_objects uint64
  // number of clones of objects
  Num_object_clones uint64
  // num_objects * num_replicas
  Num_object_copies uint64
  Num_objects_missing_on_primary uint64
  // number of objects found on no OSDs
  Num_objects_unfound uint64
  // number of objects replicated fewer times than they should be
  // (but found on at least one OSD)
  Num_objects_degraded uint64
  Num_rd uint64
  Num_rd_kb uint64
  Num_wr uint64
  Num_wr_kb uint64
}

// IOContext represents a context for performing I/O within a pool.
type IOContext struct {
    ioctx C.rados_ioctx_t
}

type Snapshot C.rados_snap_t

// Write writes len(data) bytes to the object with key oid starting at byte
// offset offset. It returns an error, if any.
func (ioctx *IOContext) Write(oid string, data []byte, offset uint64) error {
    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))

    ret := C.rados_write(ioctx.ioctx, c_oid,
        (*C.char)(unsafe.Pointer(&data[0])),
        (C.size_t)(len(data)),
        (C.uint64_t)(offset))

    if ret == 0 {
        return nil
    } else {
        return RadosError(int(ret))
    }
}

// Read reads up to len(data) bytes from the object with key oid starting at byte
// offset offset. It returns the number of bytes read and an error, if any.
func (ioctx *IOContext) Read(oid string, data []byte, offset uint64) (int, error) {
    if len(data) == 0 {
        return 0, nil
    }

    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))

    ret := C.rados_read(
        ioctx.ioctx,
        c_oid,
        (*C.char)(unsafe.Pointer(&data[0])),
        (C.size_t)(len(data)),
        (C.uint64_t)(offset))

    if ret >= 0 {
        return int(ret), nil
    } else {
        return 0, RadosError(int(ret))
    }
}

// Delete deletes the object with key oid. It returns an error, if any.
func (ioctx *IOContext) Delete(oid string) error {
    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))

    ret := C.rados_remove(ioctx.ioctx, c_oid)

    if ret == 0 {
        return nil
    } else {
        return RadosError(int(ret))
    }
}

// Truncate resizes the object with key oid to size size. If the operation
// enlarges the object, the new area is logically filled with zeroes. If the
// operation shrinks the object, the excess data is removed. It returns an
// error, if any.
func (ioctx *IOContext) Truncate(oid string, size uint64) error {
    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))

    ret := C.rados_trunc(ioctx.ioctx, c_oid, (C.uint64_t)(size))

    if ret == 0 {
        return nil
    } else {
        return RadosError(int(ret))
    }
}

// Destroy informs librados that the I/O context is no longer in use.
// Resources associated with the context may not be freed immediately, and the
// context should not be used again after calling this method.
func (ioctx *IOContext) Destroy() {
    C.rados_ioctx_destroy(ioctx.ioctx)
}

// Stat returns a set of statistics about the pool associated with this I/O
// context.
func (ioctx *IOContext) GetPoolStats() (stat PoolStat, err error) {
    c_stat := C.struct_rados_pool_stat_t{}
    ret := C.rados_ioctx_pool_stat(ioctx.ioctx, &c_stat)
    if ret < 0 {
        return PoolStat{}, RadosError(int(ret))
    } else {
        return PoolStat{
            Num_bytes: uint64(c_stat.num_bytes),
            Num_kb: uint64(c_stat.num_kb),
            Num_objects: uint64(c_stat.num_objects),
            Num_object_clones: uint64(c_stat.num_object_clones),
            Num_object_copies: uint64(c_stat.num_object_copies),
            Num_objects_missing_on_primary: uint64(c_stat.num_objects_missing_on_primary),
            Num_objects_unfound: uint64(c_stat.num_objects_unfound),
            Num_objects_degraded: uint64(c_stat.num_objects_degraded),
            Num_rd: uint64(c_stat.num_rd),
            Num_rd_kb: uint64(c_stat.num_rd_kb),
            Num_wr: uint64(c_stat.num_wr),
            Num_wr_kb: uint64(c_stat.num_wr_kb),
        }, nil
    }
}

// GetPoolName returns the name of the pool associated with the I/O context.
func (ioctx *IOContext) GetPoolName() (name string, err error) {
    buf := make([]byte, 128)
    for {
        ret := C.rados_ioctx_get_pool_name(ioctx.ioctx,
            (*C.char)(unsafe.Pointer(&buf[0])), C.unsigned(len(buf)))
        if ret == -34 { // FIXME
            buf = make([]byte, len(buf)*2)
            continue
        } else if ret < 0 {
            return "", RadosError(ret)
        }
        name = C.GoStringN((*C.char)(unsafe.Pointer(&buf[0])), ret)
        return name, nil
    }
}

// ObjectListFunc is the type of the function called for each object visited
// by ListObjects.
type ObjectListFunc func(oid string)

// ListObjects lists all of the objects in the pool associated with the I/O
// context, and called the provided listFn function for each object, passing
// to the function the name of the object.
func (ioctx *IOContext) ListObjects(listFn ObjectListFunc) error {
    var ctx C.rados_list_ctx_t
    ret := C.rados_objects_list_open(ioctx.ioctx, &ctx)
    if ret < 0 {
        return RadosError(ret)
    }
    defer func() { C.rados_objects_list_close(ctx) }()

    for {
        var c_entry *C.char
        ret := C.rados_objects_list_next(ctx, &c_entry, nil)
        if ret == -2 { // FIXME
            return nil
        } else if ret < 0 {
            return RadosError(ret)
        }
        listFn(C.GoString(c_entry))
    }

    panic("invalid state")
}

// CreateSnapshot creates a pool-wide snapshot.
func (ioctx *IOContext) CreatePoolSnapshot(name string) error {
    c_name := C.CString(name)
    defer C.free(unsafe.Pointer(c_name))
    ret := C.rados_ioctx_snap_create(ioctx.ioctx, c_name)
    if ret < 0 {
        return RadosError(ret)
    } else {
        return nil
    }
}

// RemoveSnapshot deletes a pool snapshot.
func (ioctx *IOContext) RemovePoolSnapshot(name string) error {
    c_name := C.CString(name)
    defer C.free(unsafe.Pointer(c_name))
    ret := C.rados_ioctx_snap_remove(ioctx.ioctx, c_name)
    if ret < 0 {
        return RadosError(ret)
    } else {
        return nil
    }
}

// RollbackObject restores an object to its state within a snapshot.
func (ioctx *IOContext) RollbackObject(oid, snap string) error {
    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))
    c_name := C.CString(snap)
    defer C.free(unsafe.Pointer(c_name))
    ret := C.rados_ioctx_snap_rollback(ioctx.ioctx, c_oid, c_name)
    if ret < 0 {
        return RadosError(ret)
    } else {
        return nil
    }
}

// SetReadSnapshot sets the snapshot from which reads are performed.
func (ioctx *IOContext) SetReadSnapshot(snap *Snapshot) {
    C.rados_ioctx_snap_set_read(ioctx.ioctx, C.rados_snap_t(*snap))
}


// CreateManagedSnapshot allocates an ID for a self-managed snapshot.
func (ioctx *IOContext) CreateManagedSnapshot() (*Snapshot, error) {
    var snap Snapshot
    ret := C.rados_ioctx_selfmanaged_snap_create(ioctx.ioctx, (*C.rados_snap_t)(&snap))
    if ret < 0 {
        return nil, RadosError(ret)
    } else {
        return &snap, nil
    }
}

// RemoveManagedSnapshot removes a self-managed snapshot.
func (ioctx *IOContext) RemoveManagedSnapshot(snap *Snapshot) error {
    ret := C.rados_ioctx_selfmanaged_snap_remove(ioctx.ioctx, C.rados_snap_t(*snap))
    if ret < 0 {
        return RadosError(ret)
    } else {
        return nil
    }
}

// RollbackManagedObject restores an object to its state within a self-managed
// snapshot.
func (ioctx *IOContext) RollbackManagedObject(oid string, snap *Snapshot) error {
    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))
    ret := C.rados_ioctx_selfmanaged_snap_rollback(ioctx.ioctx, c_oid, C.rados_snap_t(*snap))
    if ret < 0 {
        return RadosError(ret)
    } else {
        return nil
    }
}
