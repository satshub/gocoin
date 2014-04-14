package btc

import (
)


func nextBlock(ch *Chain, hash, header []byte, height, blen, txs uint32) {
	bh := NewUint256(hash[:])
	if _, ok := ch.BlockIndex[bh.BIdx()]; ok {
		println("nextBlock:", bh.String(), "- already in")
		return
	}
	v := new(BlockTreeNode)
	v.BlockHash = bh
	v.Height = height
	v.BlockSize = blen
	v.TxCount = txs
	copy(v.BlockHeader[:], header)
	ch.BlockIndex[v.BlockHash.BIdx()] = v
}


// Loads block index from the disk
func (ch *Chain)loadBlockIndex() {
	ch.BlockIndex = make(map[[Uint256IdxLen]byte]*BlockTreeNode, BlockMapInitLen)
	ch.BlockTreeRoot = new(BlockTreeNode)
	ch.BlockTreeRoot.BlockHash = ch.Genesis
	ch.BlockIndex[ch.Genesis.BIdx()] = ch.BlockTreeRoot


	ch.Blocks.LoadBlockIndex(ch, nextBlock)
	tlb := ch.Unspent.GetLastBlockHash()
	//println("Building tree from", len(ch.BlockIndex), "nodes")
	for _, v := range ch.BlockIndex {
		if AbortNow {
			return
		}
		if v==ch.BlockTreeRoot {
			// skip root block (should be only one)
			continue
		}
		par, ok := ch.BlockIndex[v.ParenHash().BIdx()]
		if !ok {
			panic(v.BlockHash.String()+" has no Parent "+v.ParenHash().String())
		}
		/*if par.Height+1 != v.Height {
			panic("height mismatch")
		}*/
		v.Parent = par
		v.Parent.addChild(v)
	}
	if tlb == nil {
		//println("No last block - full rescan will be needed")
		ch.BlockTreeEnd = ch.BlockTreeRoot
		return
	} else {
		var ok bool
		ch.BlockTreeEnd, ok = ch.BlockIndex[NewUint256(tlb).BIdx()]
		if !ok {
			panic("Last Block Hash not found")
		}
	}
}
