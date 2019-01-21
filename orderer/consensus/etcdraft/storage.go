
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
版权所有IBM公司。保留所有权利。

SPDX许可证标识符：Apache-2.0
**/


package etcdraft

import (
	"os"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/snap"
	"github.com/coreos/etcd/wal"
	"github.com/coreos/etcd/wal/walpb"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
)

//memoryStorage目前由etcd/raft.memoryStorage支持。这个接口是
//定义为公开FSM的依赖项，以便在
//未来。todo（jay）在需要时向该接口添加其他必要的方法
//它们在实现中，例如applysnapshot。
type MemoryStorage interface {
	raft.Storage
	Append(entries []raftpb.Entry) error
	SetHardState(st raftpb.HardState) error
	CreateSnapshot(i uint64, cs *raftpb.ConfState, data []byte) (raftpb.Snapshot, error)
	Compact(compactIndex uint64) error
	ApplySnapshot(snap raftpb.Snapshot) error
}

//raftstorage封装了ETCD/raft数据所需的存储，即内存、wal
type RaftStorage struct {
	SnapshotCatchUpEntries uint64

	lg *flogging.FabricLogger

	ram  MemoryStorage
	wal  *wal.WAL
	snap *snap.Snapshotter
}

//createStorage试图创建一个存储来保存etcd/raft数据。
//如果数据出现在指定的磁盘中，则加载它们以重建存储状态。
func CreateStorage(
	lg *flogging.FabricLogger,
	applied uint64,
	walDir string,
	snapDir string,
	ram MemoryStorage,
) (*RaftStorage, error) {

	sn, err := createSnapshotter(snapDir)
	if err != nil {
		return nil, err
	}

	snapshot, err := sn.Load()
	if err != nil {
		if err == snap.ErrNoSnapshot {
			lg.Debugf("No snapshot found at %s", snapDir)
		} else {
			return nil, errors.Errorf("failed to load snapshot: %s", err)
		}
	} else {
//发现快照
		lg.Debugf("Loaded snapshot at Term %d and Index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
	}

	w, err := createWAL(lg, walDir, applied, snapshot)
	if err != nil {
		return nil, err
	}

	_, st, ents, err := w.ReadAll()
	if err != nil {
		return nil, errors.Errorf("failed to read WAL: %s", err)
	}

	if snapshot != nil {
		lg.Debugf("Applying snapshot to raft MemoryStorage")
		if err := ram.ApplySnapshot(*snapshot); err != nil {
			return nil, errors.Errorf("Failed to apply snapshot to memory: %s", err)
		}
	}

	lg.Debugf("Setting HardState to {Term: %d, Commit: %d}", st.Term, st.Commit)
ram.SetHardState(st) //memoryStorage.setHardstate始终返回零

	lg.Debugf("Appending %d entries to memory storage", len(ents))
ram.Append(ents) //memoryStorage.append始终返回nil

	return &RaftStorage{lg: lg, ram: ram, wal: w, snap: sn}, nil
}

func createSnapshotter(snapDir string) (*snap.Snapshotter, error) {
	if err := os.MkdirAll(snapDir, os.ModePerm); err != nil {
		return nil, errors.Errorf("failed to mkdir '%s' for snapshot: %s", snapDir, err)
	}

	return snap.New(snapDir), nil

}

func createWAL(lg *flogging.FabricLogger, walDir string, applied uint64, snapshot *raftpb.Snapshot) (*wal.WAL, error) {
	hasWAL := wal.Exist(walDir)
	if !hasWAL && applied != 0 {
		return nil, errors.Errorf("applied index is not zero but no WAL data found")
	}

	if !hasWAL {
		lg.Infof("No WAL data found, creating new WAL at path '%s'", walDir)
//Todo（Jay_Guo）添加元数据，以便在需要时与Wal一起持久化。
//用例可以是新节点上的数据转储和恢复。
		w, err := wal.Create(walDir, nil)
		if err == os.ErrExist {
			lg.Fatalf("programming error, we've just checked that WAL does not exist")
		}

		if err != nil {
			return nil, errors.Errorf("failed to initialize WAL: %s", err)
		}

		if err = w.Close(); err != nil {
			return nil, errors.Errorf("failed to close the WAL just created: %s", err)
		}
	} else {
		lg.Infof("Found WAL data at path '%s', replaying it", walDir)
	}

	walsnap := walpb.Snapshot{}
	if snapshot != nil {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}

	lg.Debugf("Loading WAL at Term %d and Index %d", walsnap.Term, walsnap.Index)
	w, err := wal.Open(walDir, walsnap)
	if err != nil {
		return nil, errors.Errorf("failed to open existing WAL: %s", err)
	}

	return w, nil
}

//快照返回存储在内存中的最新快照
func (rs *RaftStorage) Snapshot() raftpb.Snapshot {
sn, _ := rs.ram.Snapshot() //快照始终返回零错误
	return sn
}

//保存保存的ETCD/RAFT数据
func (rs *RaftStorage) Store(entries []raftpb.Entry, hardstate raftpb.HardState, snapshot raftpb.Snapshot) error {
	if err := rs.wal.Save(hardstate, entries); err != nil {
		return err
	}

	if !raft.IsEmptySnap(snapshot) {
		if err := rs.saveSnap(snapshot); err != nil {
			return err
		}

		if err := rs.ram.ApplySnapshot(snapshot); err != nil {
			if err == raft.ErrSnapOutOfDate {
				rs.lg.Warnf("Attempted to apply out-of-date snapshot at Term %d and Index %d",
					snapshot.Metadata.Term, snapshot.Metadata.Index)
			} else {
				rs.lg.Fatalf("Unexpected programming error: %s", err)
			}
		}
	}

	if err := rs.ram.Append(entries); err != nil {
		return err
	}

	return nil
}

func (rs *RaftStorage) saveSnap(snap raftpb.Snapshot) error {
//必须先将快照索引保存到wal，然后才能保存
//快照保持不变，我们只打开
//以前保存的快照索引。
	walsnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term:  snap.Metadata.Term,
	}

	rs.lg.Debugf("Saving snapshot to WAL")
	if err := rs.wal.SaveSnapshot(walsnap); err != nil {
		return errors.Errorf("failed to save snapshot to WAL: %s", err)
	}

	rs.lg.Debugf("Saving snapshot to disk")
	if err := rs.snap.SaveSnap(snap); err != nil {
		return errors.Errorf("failed to save snapshot to disk: %s", err)
	}

	rs.lg.Debugf("Releasing lock to wal files prior to %d", snap.Metadata.Index)
	if err := rs.wal.ReleaseLockTo(snap.Metadata.Index); err != nil {
		return err
	}

	return nil
}

//takesnapshot从memorystorage在索引I处获取快照，并将其保存到wal和disk。
func (rs *RaftStorage) TakeSnapshot(i uint64, cs *raftpb.ConfState, data []byte) error {
	rs.lg.Debugf("Creating snapshot at index %d from MemoryStorage", i)
	snap, err := rs.ram.CreateSnapshot(i, cs, data)
	if err != nil {
		return errors.Errorf("failed to create snapshot from MemoryStorage: %s", err)
	}

	if err = rs.saveSnap(snap); err != nil {
		return err
	}

//在内存中保留一些条目，以便缓慢的追随者赶上
	if i > rs.SnapshotCatchUpEntries {
		compacti := i - rs.SnapshotCatchUpEntries
		rs.lg.Debugf("Purging in-memory raft entries prior to %d", compacti)
		if err = rs.ram.Compact(compacti); err != nil {
			if err == raft.ErrCompacted {
				rs.lg.Warnf("Raft entries prior to %d are already purged", compacti)
			} else {
				rs.lg.Fatalf("Failed to purg raft entries: %s", err)
			}
		}
	}

	rs.lg.Infof("Snapshot is taken at index %d", i)
	return nil
}

//ApplySnapshot将快照应用于本地内存存储
func (rs *RaftStorage) ApplySnapshot(snap raftpb.Snapshot) {
	if err := rs.ram.ApplySnapshot(snap); err != nil {
		if err == raft.ErrSnapOutOfDate {
			rs.lg.Warnf("Attempted to apply out-of-date snapshot at Term %d and Index %d",
				snap.Metadata.Term, snap.Metadata.Index)
		} else {
			rs.lg.Fatalf("Unexpected programming error: %s", err)
		}
	}
}

//关闭关闭存储
func (rs *RaftStorage) Close() error {
	if err := rs.wal.Close(); err != nil {
		return err
	}

	return nil
}
