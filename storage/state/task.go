package state

type TaskStatus uint8

const (
	TaskStatusUnknown TaskStatus = 0

	// Publish 已完成：任务已在 taskShard 创建并注入 RewardPool
	TaskStatusPublished TaskStatus = 1

	// Bidding：至少有过一次投标
	TaskStatusBidding TaskStatus = 2

	// Settling：结算中
	TaskStatusSettling TaskStatus = 3

	// Settled：已结算
	TaskStatusSettled TaskStatus = 4

	// Cancelled：已取消
	TaskStatusCancelled TaskStatus = 5
)

type Task struct {
	TaskID    string
	Requester string

	Title       string
	Description string

	Status TaskStatus

	RewardPool uint64              // publish 锁仓
	Deposits   map[[20]byte]uint64 // workerAddr -> deposit

	Winner  string
	Settled bool
}

func (s *ShardState) GetTask(taskId32 [32]byte) *Task {
	task, ok := s.Tasks[taskId32]
	if !ok {
		task = &Task{}
		s.Tasks[taskId32] = task
	}
	return task
}

// cloneTask 深复制 Task，确保修改副本不会影响原始 Task
func cloneTask(t *Task) *Task {
	if t == nil {
		return nil
	}
	cp := *t
	if t.Deposits != nil {
		cp.Deposits = make(map[[20]byte]uint64, len(t.Deposits))
		for k, v := range t.Deposits {
			cp.Deposits[k] = v
		}
	}
	return &cp
}
