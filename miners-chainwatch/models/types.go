package models

const (
	ProveCommitSector = "ProveCommitSector"
	PreCommitSector = 	"PreCommitSector"
)

type TipSet struct {
	Cid    string
	Height int64
	RawBytePower     string
	QualityAdjPower  string
	ThisEpochReward  string
	Blocks []Block
	MinerInfo []MinerInfo
	Messages []Message
}

type Block struct {
	Cid    string
	Miner  string
	Parents []string
	ParentWeight string
	ParentStateRoot string
	Height int64
	Timestamp uint64
	ParentBaseFee string
	WinCount int64
}

type Message struct {
	Cid            string
	Version        uint64
	To             string
	From           string
	Nonce          uint64
	Value          string
	GasLimit       int64
	GasFeeCap      string
	GasPremium     string
	Method         string
	GasUse         int64
	ExitCode       string
}

type MinerInfo struct {
	Address          string
	RawBytePower     string
	QualityAdjPower  string
	AvailableBalance string
	Balance          string
}

type Statistics struct {
	StatisticsTime   string
	StartHeight      int64
	EndHeight        int64
	Address 		 string
	PledgeFonds      string
	TotalWinCount    int64
	TotalWinNumber   int64
	TotalBlockReward string
}