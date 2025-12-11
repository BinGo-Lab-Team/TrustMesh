package consensus

// 关键常量
const (
	// ScoreBurrs 广播阈值
	ScoreBurrs = 10
	// DiffuseHop 传播节点数量（具体逻辑参考代码）
	DiffuseHop = 100

	// MaxReputation 最大信誉值
	MaxReputation = 10_000
	// MinReputation 最小信誉值
	MinReputation = 0

	// MaxScore 最大分数
	MaxScore = 10_000
	// MinScore 最小分数
	MinScore = 0
	// MyScore 给自己的评分
	MyScore = 5_000
)

const (
	// RATEV1Domain 打分签名 Tag
	RATEV1Domain uint32 = 0x48783BC2
	// GUARANTEEV1Domain 担保签名 Tag
	GUARANTEEV1Domain uint32 = 0x6DB7008D
	// PROPOSERV1Domain 提案者签名
	PROPOSERV1Domain uint32 = 0x3A174310
)
