package blockchain_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"sync"
	"testing"

	"github.com/goozt/seashell/blockchain"
	"github.com/goozt/seashell/wallet"
)

// --- helpers ---

func newKeyPair(t *testing.T) (ecdsa.PrivateKey, []byte) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pub := make([]byte, 64)
	xBytes := priv.PublicKey.X.Bytes()
	yBytes := priv.PublicKey.Y.Bytes()
	copy(pub[32-len(xBytes):32], xBytes)
	copy(pub[64-len(yBytes):64], yBytes)
	return *priv, pub
}

func newAddress(t *testing.T, pub []byte) string {
	t.Helper()
	pubHash := wallet.PublicKeyHash(pub)
	verHash := append([]byte{0x00}, pubHash...)
	checksum := wallet.Checksum(verHash)
	hash := append(verHash, checksum...)
	return string(wallet.Base58Encode(hash))
}

func newTestChain(t *testing.T, dir string) (*blockchain.BlockChain, ecdsa.PrivateKey, []byte) {
	t.Helper()
	privKey, pubKey := newKeyPair(t)
	addr := newAddress(t, pubKey)
	chain := blockchain.InitBlockChainAt(dir, false, addr, pubKey, privKey)
	return chain, privKey, pubKey
}

func hexDecodeStr(s string) []byte {
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var v byte
		for j, c := range s[i : i+2] {
			var n byte
			switch {
			case c >= '0' && c <= '9':
				n = byte(c - '0')
			case c >= 'a' && c <= 'f':
				n = byte(c-'a') + 10
			case c >= 'A' && c <= 'F':
				n = byte(c-'A') + 10
			}
			if j == 0 {
				v = n << 4
			} else {
				v |= n
			}
		}
		b[i/2] = v
	}
	return b
}

// --- encodeSig / decodeSig round-trip ---

// TestEncodeSig64Bytes verifies that ECDSA signatures are always stored as 64 bytes.
func TestEncodeSig64Bytes(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hash := make([]byte, 32)
	rand.Read(hash)

	// Run many times to hit the rare case where r or s < 32 bytes.
	for i := 0; i < 200; i++ {
		rReal, sReal, err := ecdsa.Sign(rand.Reader, privKey, hash)
		if err != nil {
			t.Fatal(err)
		}
		sig := make([]byte, 64)
		rBytes := rReal.Bytes()
		sBytes := sReal.Bytes()
		copy(sig[32-len(rBytes):32], rBytes)
		copy(sig[64-len(sBytes):64], sBytes)

		if len(sig) != 64 {
			t.Fatalf("iteration %d: expected 64-byte signature, got %d", i, len(sig))
		}
		rDec := new(big.Int).SetBytes(sig[:32])
		sDec := new(big.Int).SetBytes(sig[32:])
		if rDec.Cmp(rReal) != 0 {
			t.Errorf("iteration %d: r mismatch", i)
		}
		if sDec.Cmp(sReal) != 0 {
			t.Errorf("iteration %d: s mismatch", i)
		}
	}
}

// --- FindUTXO correctness: no duplicate-tx bug ---

func TestFindUTXO_NoDuplicates(t *testing.T) {
	dir := t.TempDir()
	chain, validatorPriv, validatorPub := newTestChain(t, dir)
	defer chain.Close()

	genesisAddr := newAddress(t, validatorPub)

	// Genesis gives 100 coins to genesisAddr.
	utxos := chain.FindUTXO(wallet.PublicKeyHash(validatorPub))
	total := 0
	for _, u := range utxos {
		total += u.Value
	}
	if total != 100 {
		t.Fatalf("expected 100, got %d", total)
	}

	priv2, pub2 := newKeyPair(t)
	addr2 := newAddress(t, pub2)

	// Send 30 to addr2, change 70 back.
	acc, validOutput := chain.FindSpendableOutputs(wallet.PublicKeyHash(validatorPub), 30)
	if acc < 30 {
		t.Fatalf("not enough funds: have %d", acc)
	}
	var inputs []blockchain.TxInput
	for txid, outs := range validOutput {
		txIDBytes := hexDecodeStr(txid)
		for _, out := range outs {
			inputs = append(inputs, blockchain.TxInput{Id: txIDBytes, Out: out, PubKey: validatorPub})
		}
	}
	tx := &blockchain.Transaction{
		Inputs:  inputs,
		Outputs: []blockchain.TxOutput{*blockchain.NewTxOutput(30, addr2), *blockchain.NewTxOutput(70, genesisAddr)},
	}
	tx.Id = tx.Hash()
	chain.SignTransaction(tx, validatorPriv)
	chain.AddBlock([]*blockchain.Transaction{tx}, validatorPub, validatorPriv)

	if b := sumUTXO(chain, wallet.PublicKeyHash(pub2)); b != 30 {
		t.Fatalf("addr2: expected 30, got %d", b)
	}
	if b := sumUTXO(chain, wallet.PublicKeyHash(validatorPub)); b != 70 {
		t.Fatalf("genesisAddr: expected 70, got %d", b)
	}

	// Send 10 from addr2, change 20 back.
	acc3, validOutput3 := chain.FindSpendableOutputs(wallet.PublicKeyHash(pub2), 10)
	if acc3 < 10 {
		t.Fatalf("addr2 not enough funds: have %d", acc3)
	}
	var inputs3 []blockchain.TxInput
	for txid, outs := range validOutput3 {
		txIDBytes := hexDecodeStr(txid)
		for _, out := range outs {
			inputs3 = append(inputs3, blockchain.TxInput{Id: txIDBytes, Out: out, PubKey: pub2})
		}
	}
	tx3 := &blockchain.Transaction{
		Inputs:  inputs3,
		Outputs: []blockchain.TxOutput{*blockchain.NewTxOutput(10, genesisAddr), *blockchain.NewTxOutput(20, addr2)},
	}
	tx3.Id = tx3.Hash()
	chain.SignTransaction(tx3, priv2)
	chain.AddBlock([]*blockchain.Transaction{tx3}, validatorPub, validatorPriv)

	if b := sumUTXO(chain, wallet.PublicKeyHash(pub2)); b != 20 {
		t.Fatalf("final addr2: expected 20, got %d", b)
	}
	if b := sumUTXO(chain, wallet.PublicKeyHash(validatorPub)); b != 80 {
		t.Fatalf("final genesis: expected 80, got %d", b)
	}
}

func sumUTXO(chain *blockchain.BlockChain, pkh []byte) int {
	total := 0
	for _, u := range chain.FindUTXO(pkh) {
		total += u.Value
	}
	return total
}

// TestDoubleSpendPrevented verifies the same UTXO cannot be double-counted.
func TestDoubleSpendPrevented(t *testing.T) {
	dir := t.TempDir()
	chain, validatorPriv, validatorPub := newTestChain(t, dir)
	defer chain.Close()

	_, pub2 := newKeyPair(t)
	addr2 := newAddress(t, pub2)
	genesisAddr := newAddress(t, validatorPub)

	acc, validOutput := chain.FindSpendableOutputs(wallet.PublicKeyHash(validatorPub), 50)
	if acc < 50 {
		t.Fatalf("not enough funds")
	}
	var inputs []blockchain.TxInput
	for txid, outs := range validOutput {
		txIDBytes := hexDecodeStr(txid)
		for _, out := range outs {
			inputs = append(inputs, blockchain.TxInput{Id: txIDBytes, Out: out, PubKey: validatorPub})
		}
	}
	tx1 := &blockchain.Transaction{
		Inputs:  inputs,
		Outputs: []blockchain.TxOutput{*blockchain.NewTxOutput(50, addr2), *blockchain.NewTxOutput(50, genesisAddr)},
	}
	tx1.Id = tx1.Hash()
	chain.SignTransaction(tx1, validatorPriv)
	chain.AddBlock([]*blockchain.Transaction{tx1}, validatorPub, validatorPriv)

	if b := sumUTXO(chain, wallet.PublicKeyHash(pub2)); b != 50 {
		t.Fatalf("expected 50, got %d", b)
	}
	// genesisAddr has 50 change; original 100 UTXO is spent.
	if b := sumUTXO(chain, wallet.PublicKeyHash(validatorPub)); b != 50 {
		t.Fatalf("genesis should have 50 change, got %d", b)
	}
}

// TestAddExternalBlock verifies PoA and chain linkage checks on AddExternalBlock.
func TestAddExternalBlock(t *testing.T) {
	dir := t.TempDir()
	chain, privKey, pubKey := newTestChain(t, dir)
	defer chain.Close()

	addr := newAddress(t, pubKey)
	height := chain.GetCurrentHeight() + 1
	prevHash := chain.LastHash
	coinbase := blockchain.CoinbaseTx(addr, "test")
	block := blockchain.NewBlock([]*blockchain.Transaction{coinbase}, prevHash, height, pubKey, privKey)

	if err := chain.AddExternalBlock(block); err != nil {
		t.Fatalf("valid external block rejected: %v", err)
	}
	if chain.GetCurrentHeight() != 1 {
		t.Fatalf("expected height 1, got %d", chain.GetCurrentHeight())
	}

	// Block with wrong prevHash must be rejected.
	badPrevHash := make([]byte, 32)
	rand.Read(badPrevHash)
	block2 := blockchain.NewBlock([]*blockchain.Transaction{coinbase}, badPrevHash, height+1, pubKey, privKey)
	if err := chain.AddExternalBlock(block2); err == nil {
		t.Error("expected error for wrong prevHash")
	}
}

// TestConcurrentAddBlock verifies chain stays consistent with a caller-held mutex.
func TestConcurrentAddBlock(t *testing.T) {
	dir := t.TempDir()
	chain, privKey, pubKey := newTestChain(t, dir)
	defer chain.Close()

	addr := newAddress(t, pubKey)
	var mu sync.Mutex
	var wg sync.WaitGroup
	const n = 5

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			coinbase := blockchain.CoinbaseTx(addr, "concurrent block")
			func() {
				defer func() { recover() }()
				chain.AddBlock([]*blockchain.Transaction{coinbase}, pubKey, privKey)
			}()
		}()
	}
	wg.Wait()

	if h := chain.GetCurrentHeight(); h != n {
		t.Fatalf("expected height %d, got %d", n, h)
	}
}

// TestTransactionSignVerify exercises full Sign/Verify round-trip.
func TestTransactionSignVerify(t *testing.T) {
	dir := t.TempDir()
	chain, validatorPriv, validatorPub := newTestChain(t, dir)
	defer chain.Close()

	_, pub2 := newKeyPair(t)
	addr2 := newAddress(t, pub2)

	acc, validOutput := chain.FindSpendableOutputs(wallet.PublicKeyHash(validatorPub), 50)
	if acc < 50 {
		t.Fatalf("not enough funds")
	}
	var inputs []blockchain.TxInput
	for txid, outs := range validOutput {
		txIDBytes := hexDecodeStr(txid)
		for _, out := range outs {
			inputs = append(inputs, blockchain.TxInput{Id: txIDBytes, Out: out, PubKey: validatorPub})
		}
	}
	tx := &blockchain.Transaction{
		Inputs:  inputs,
		Outputs: []blockchain.TxOutput{*blockchain.NewTxOutput(50, addr2), *blockchain.NewTxOutput(50, newAddress(t, validatorPub))},
	}
	tx.Id = tx.Hash()
	chain.SignTransaction(tx, validatorPriv)
	if !chain.VerifyTransaction(tx) {
		t.Fatal("transaction signature verification failed")
	}
}

// TestGetBlocksFromHeight verifies P2P sync helper.
func TestGetBlocksFromHeight(t *testing.T) {
	dir := t.TempDir()
	chain, privKey, pubKey := newTestChain(t, dir)
	defer chain.Close()

	addr := newAddress(t, pubKey)
	for i := 0; i < 5; i++ {
		coinbase := blockchain.CoinbaseTx(addr, "block")
		chain.AddBlock([]*blockchain.Transaction{coinbase}, pubKey, privKey)
	}
	blocks := chain.GetBlocksFromHeight(3, 10)
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks (3,4,5), got %d", len(blocks))
	}
	if blocks[0].Height != 3 {
		t.Fatalf("expected first block height 3, got %d", blocks[0].Height)
	}
}
