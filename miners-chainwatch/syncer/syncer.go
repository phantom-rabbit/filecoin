package syncer

import (
	"context"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"sort"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	logging "github.com/ipfs/go-log/v2"

	"github.com/phantom-rabbit/filecoin/miners-chainwatch/models"
	"github.com/phantom-rabbit/filecoin/util"
)

const (
	chainData = "chain-data"
)

var log = logging.Logger("syncer")

var (
	startChainEpoch = int64(148888)
	endCHainEpoch = int64(-1)
)

type Syncer struct {
	miners             []address.Address
	maxbatch           int
	node               api.FullNode
}

func NewSyncer(minersAddress []string, maxbatch, startEpoch, endEpoch int, node api.FullNode) (*Syncer, error) {
	miners := make([]address.Address, len(minersAddress))
	for _, addr := range minersAddress {
		m, err := address.NewFromString(addr)
		if err != nil {
			log.Error("parse miner address fail: %s", addr)
			return nil, err
		}
		miners = append(miners, m)
	}

	if startEpoch > 0 {
		startChainEpoch = int64(startEpoch)
		endCHainEpoch = int64(endEpoch)
	}

	return &Syncer{
		miners: miners,
		maxbatch: maxbatch,
		node: node,
	}, nil

}

func (s *Syncer) Start(ctx context.Context)  {
	if err := logging.SetLogLevel("syncer", "info"); err != nil {
		log.Fatal(err)
	}
	log.Info("start epoch:", startChainEpoch)

	go s.job()

	maxGorouting := make(chan struct{}, s.maxbatch)
	height := startChainEpoch
	for {
		head, err := s.node.ChainHead(ctx)
		if err != nil {
			log.Fatal(err)
		}

		var localTipset models.TipSet

		err = models.FindLastTipset(chainData, nil,  &localTipset)
		if err == nil {
			height = localTipset.Height
		}

		for height <= (int64(head.Height()) - int64(miner.ChainFinality)) {
			if height > endCHainEpoch {
				log.Info("sync Chain Data down!!!")
				break
			}
			maxGorouting <- struct{}{}
			go func(h int64) {
				s.syncChainData(ctx, abi.ChainEpoch(h))
				<- maxGorouting
			}(height)
			height ++
		}
		time.Sleep(30 * 2 * time.Second)
	}
}

func (s *Syncer)job ()  {
	start := startChainEpoch
	section := int64(builtin.EpochsInDay)
	timer := time.NewTimer(time.Minute * 10)
	defer timer.Stop()
	for {
		<- timer.C

		var localTipset models.TipSet
		err := models.FindLastTipset(chainData, nil,  &localTipset)
		if err != nil {
			log.Fatal(err)
		}

		for {
			if start > localTipset.Height {
				timer.Reset(time.Hour + time.Minute * 10 )
				break
			}

			fmt.Println(start, start+section)
			var ti []models.TipSet
			err := models.FindAllSort(chainData, bson.M{"height": bson.M{"$gte": start, "$lt": start + section}}, &ti)
			if err != nil {
				log.Fatal(err)
			}
			sort.Slice(ti, func(i, j int) bool {
				return ti[i].Height < ti[j].Height
			})

			fmt.Println("start:", ti[0].Height, "end:", ti[len(ti) -1].Height, "total:", len(ti))

			for _, m := range s.miners {
				s.StatisticsMinerInfo(m.String() , ti)
			}
			start += section
		}
	}

}

func (s *Syncer) syncChainData(ctx context.Context, height abi.ChainEpoch) {
	startTime := time.Now().UnixNano()
	defer log.Info("sync chain data which height:", height, time.Now().UnixNano() - startTime)

	t, err := s.node.ChainGetTipSetByHeight(ctx, height, types.EmptyTSK)
	if err != nil {
		log.Fatal(err)
	}

	tNextHeight := height + 1
	var tNext *types.TipSet
	for {
		tNext, err = s.node.ChainGetTipSetByHeight(ctx, tNextHeight, types.EmptyTSK)
		if err != nil {
			log.Fatal(err)
		}
		if tNext.Height() == height {
			tNextHeight ++
		} else {
			break
		}
	}

	recs, err := s.node.ChainGetParentReceipts(ctx, tNext.Blocks()[0].Cid())
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := s.node.ChainGetParentMessages(ctx, tNext.Blocks()[0].Cid())
	if err != nil {
		log.Error(err)
		return
	}

	blockMsgs := make([]models.Message, 0)
	for i, msg := range msgs {
		var method string
		switch uint64(msg.Message.Method) {
		case 0:
			method = "Send"
		case 1:
			method = "MethodConstructor"
		default:
			a, err := s.node.StateGetActor(ctx, msg.Message.To, types.EmptyTSK)
			if err != nil {
				a, err = s.node.StateGetActor(ctx, msg.Message.To, t.Key())
				if err != nil {
					a, err = s.node.StateGetActor(ctx, msg.Message.To, tNext.Key())
					if err != nil {
						log.Fatal(err, msg.Cid)
					}
				}
			}

			method = util.LotusMethodToStr(a.Code, uint64(msg.Message.Method))
		}

		blockMsgs = append(blockMsgs, models.Message{
			Cid:       msg.Message.Cid().String(),
			Version:   msg.Message.Version,
			To:        msg.Message.To.String(),
			From:      msg.Message.From.String(),
			Nonce:     msg.Message.Nonce,
			Value:     msg.Message.Value.String(),
			GasLimit:  msg.Message.GasLimit,
			GasFeeCap: msg.Message.GasFeeCap.String(),
			GasPremium:msg.Message.GasPremium.String(),
			Method:    method,
			GasUse:    recs[i].GasUsed,
			ExitCode:  recs[i].ExitCode.String(),
		})
	}

	block := make([]models.Block, 0)
	for _, cid := range t.Cids() {
		chainBlock, err := s.node.ChainGetBlock(ctx, cid)
		if err != nil {
			log.Fatal(err)
		}

		ps := make([]string, 0)
		for _, parent := range chainBlock.Parents {
			ps = append(ps, parent.String())
		}

		block = append(block, models.Block{
			Cid:             cid.String(),
			Miner:           chainBlock.Miner.String(),
			Parents:         ps,
			ParentWeight:    chainBlock.ParentWeight.String(),
			ParentStateRoot: chainBlock.ParentStateRoot.String(),
			Height:          int64(chainBlock.Height),
			Timestamp:       chainBlock.Timestamp,
			ParentBaseFee:   chainBlock.ParentBaseFee.String(),
			WinCount:        chainBlock.ElectionProof.WinCount,
		})
	}

	localMiner := make([]models.MinerInfo, 0)
	var rawbytepower abi.StoragePower
	var qualityAdjPower abi.StoragePower

	for _, miner := range s.miners {
		balance, err := s.node.WalletBalance(ctx, miner)
		if err != nil {
			log.Fatal(err)
		}
		availableBalance, err := s.node.StateMinerAvailableBalance(ctx, miner, t.Key())
		if err != nil {
			availableBalance = types.FromFil(0)
		}
		power, err := s.node.StateMinerPower(ctx, miner, t.Key())
		if err != nil {
			log.Fatal(err)
		}

		localMiner = append(localMiner, models.MinerInfo{
			Address:          miner.String(),
			RawBytePower:     power.MinerPower.RawBytePower.String(),
			QualityAdjPower:  power.MinerPower.QualityAdjPower.String(),
			AvailableBalance: availableBalance.String(),
			Balance:          balance.String(),

		})
		rawbytepower = power.TotalPower.RawBytePower
		rawbytepower = power.TotalPower.RawBytePower
	}

	rewoardAddress, err := address.NewFromString("f02")
	if err != nil {
		log.Fatal(err)
	}
	state, err := s.node.StateReadState(ctx, rewoardAddress, t.Key())
	if err != nil {
		log.Fatal(err)
	}

	epochReward := state.State.(map[string]interface{})["ThisEpochReward"]
	fromString, err := big.FromString(epochReward.(string))
	if err != nil {
		log.Fatal(err)
	}

	thisEpochReward := big.Div(fromString, big.NewInt(builtin.ExpectedLeadersPerEpoch))

	tipset := models.TipSet{
		Cid: tNext.ParentState().String(),
		Height: int64(t.Height()),
		Blocks: block,
		RawBytePower: rawbytepower.String(),
		QualityAdjPower: qualityAdjPower.String(),
		ThisEpochReward: thisEpochReward.String(),
		MinerInfo: localMiner,
		Messages: blockMsgs,
	}

	selector := bson.M{"height": tipset.Height}

	err = models.Upsert(chainData, selector, bson.M{"$set":tipset})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Syncer) StatisticsMinerInfo(minerAddress string, tipSets []models.TipSet)  {
	pledgeFonds:=    big.Zero()
	totalReward:=    big.Zero()
	totalWincount:=  big.Zero()
	totalWinNumber:= 0

	for _, t := range tipSets {
		for _, b := range t.Blocks {
			if b.Miner == minerAddress {
				epochReward, _ := big.FromString(t.ThisEpochReward)
				reward := big.Mul(big.NewInt(b.WinCount), epochReward)
				totalReward = big.Add(totalReward, reward)
				totalWincount = big.Add(totalWincount, big.NewInt(b.WinCount))
				totalWinNumber ++
			}
		}

		for _, msg := range t.Messages {
			if msg.To == minerAddress {
				switch msg.Method {
				case models.PreCommitSector:
					value, _ := big.FromString(msg.Value)
					pledgeFonds = big.Add(pledgeFonds, value)
				case models.ProveCommitSector:
					value, _ := big.FromString(msg.Value)
					pledgeFonds = big.Add(pledgeFonds, value)
				}
			}
		}
	}

	pledgeFil, _ := types.ParseFIL(pledgeFonds.String() + "attofil")
	totalRewardFil, _ := types.ParseFIL(totalReward.String() + "attofil")

	statistics := models.Statistics{
		StatisticsTime: time.Unix(int64(tipSets[0].Blocks[0].Timestamp), 0).Format("2006-01-02 15:04:05"),
		StartHeight:      tipSets[0].Height,
		EndHeight:        tipSets[len(tipSets)-1].Height,
		Address:          minerAddress,
		PledgeFonds:      pledgeFil.String(),
		TotalWinCount:    totalWincount.Int64(),
		TotalWinNumber:   int64(totalWinNumber),
		TotalBlockReward: totalRewardFil.String(),
	}

	err := models.Insert("statistics", statistics)
	if err != nil {
		log.Fatal(err)
	}
}