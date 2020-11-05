package util

import (
	"reflect"

	//builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/ipfs/go-cid"
)

func LotusMethodToStr(code cid.Cid, methodNum uint64) string {
	if methodNum == uint64(builtin2.MethodSend ){
		return "Send"
	}

	if methodNum == uint64(builtin2.MethodConstructor ){
		return "MethodConstructor"
	}

	if code == builtin2.AccountActorCodeID {
		return methodStr(builtin2.MethodsAccount, methodNum)
	}

	if code == builtin2.InitActorCodeID {
		return methodStr(builtin2.MethodsInit, methodNum)
	}

	if code == builtin2.CronActorCodeID {
		return methodStr(builtin2.MethodsCron, methodNum)
	}

	if code == builtin2.RewardActorCodeID {
		return methodStr(builtin2.MethodsReward, methodNum)
	}

	if code == builtin2.MultisigActorCodeID {
		return methodStr(builtin2.MethodsMultisig, methodNum)
	}

	if code == builtin2.PaymentChannelActorCodeID {
		return methodStr(builtin2.MethodsPaych, methodNum)
	}

	if code == builtin2.StorageMarketActorCodeID {
		return methodStr(builtin2.MethodsMarket, methodNum)
	}

	if code == builtin2.StoragePowerActorCodeID {
		return methodStr(builtin2.MethodsPower, methodNum)
	}

	if code == builtin2.StorageMinerActorCodeID {
		return methodStr(builtin2.MethodsMiner, methodNum)
	}

	if code == builtin2.VerifiedRegistryActorCodeID {
		return methodStr(builtin2.MethodsVerifiedRegistry, methodNum)
	}

	return "UNKNOW ACTOR CODE"
}

func methodStr(v interface{}, methodNum uint64) string {
	rt := reflect.ValueOf(v)
	for i :=0; i < rt.NumField(); i ++ {
		if rt.Field(i).Uint() == methodNum {
			return rt.Type().Field(i).Name
		}
	}

	return "UNKNOW METHOD"
}