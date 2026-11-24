package clients

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// HLSignature is an Ethereum ECDSA signature in r/s/v format as Hyperliquid expects.
type HLSignature struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}

// hlDomainSeparator is the precomputed EIP-712 domain separator for Hyperliquid.
// Domain: { name: "Exchange", version: "1", chainId: 1337, verifyingContract: 0x0...0 }
// ChainId 1337 is hardcoded by Hyperliquid for L1 action signing on both mainnet and testnet.
var hlDomainSeparator = computeHLDomainSeparator()

func computeHLDomainSeparator() []byte {
	domainTypeHash := crypto.Keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := crypto.Keccak256([]byte("Exchange"))
	versionHash := crypto.Keccak256([]byte("1"))

	chainId := make([]byte, 32)
	big.NewInt(1337).FillBytes(chainId)

	verifyingContract := make([]byte, 32) // 0x0000...0000

	encoded := make([]byte, 0, 32*5)
	encoded = append(encoded, domainTypeHash...)
	encoded = append(encoded, nameHash...)
	encoded = append(encoded, versionHash...)
	encoded = append(encoded, chainId...)
	encoded = append(encoded, verifyingContract...)

	return crypto.Keccak256(encoded)
}

var hlAgentTypeHash = crypto.Keccak256([]byte("Agent(string source,bytes32 connectionId)"))

// hlSignL1Action signs a Hyperliquid L1 action using EIP-712 with chainId=1337.
//
// The signing procedure mirrors the official Python SDK:
//  1. msgpack-encode the action
//  2. append nonce (8-byte big-endian) + vault flag + vault address bytes
//  3. keccak256 → action hash
//  4. EIP-712 sign phantom Agent{source, connectionId=action_hash}
func hlSignL1Action(
	privateKeyHex string,
	action hlOrderedMap,
	vaultAddress *string,
	nonce int64,
	isMainnet bool,
) (*HLSignature, error) {
	actionBytes, err := hlMsgpackEncode(action)
	if err != nil {
		return nil, fmt.Errorf("hlSignL1Action: msgpack encode: %w", err)
	}

	var payload []byte
	payload = append(payload, actionBytes...)

	nonceBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, uint64(nonce))
	payload = append(payload, nonceBuf...)

	if vaultAddress == nil {
		payload = append(payload, 0x00)
	} else {
		payload = append(payload, 0x01)
		addrBytes, err := hex.DecodeString(strings.TrimPrefix(*vaultAddress, "0x"))
		if err != nil {
			return nil, fmt.Errorf("hlSignL1Action: invalid vault address %q: %w", *vaultAddress, err)
		}
		payload = append(payload, addrBytes...)
	}

	actionHash := crypto.Keccak256(payload)

	// "a" = mainnet, "b" = testnet
	source := "a"
	if !isMainnet {
		source = "b"
	}

	sourceHash := crypto.Keccak256([]byte(source))
	structEncoded := make([]byte, 0, 96)
	structEncoded = append(structEncoded, hlAgentTypeHash...)
	structEncoded = append(structEncoded, sourceHash...)
	structEncoded = append(structEncoded, actionHash...)
	structHash := crypto.Keccak256(structEncoded)

	digestInput := make([]byte, 0, 66)
	digestInput = append(digestInput, 0x19, 0x01)
	digestInput = append(digestInput, hlDomainSeparator...)
	digestInput = append(digestInput, structHash...)
	digest := crypto.Keccak256(digestInput)

	privKey, err := hlParsePrivateKey(privateKeyHex)
	if err != nil {
		return nil, err
	}

	sig, err := crypto.Sign(digest, privKey)
	if err != nil {
		return nil, fmt.Errorf("hlSignL1Action: sign: %w", err)
	}

	// sig is [R(32) || S(32) || V(1)] where V is 0 or 1 (recovery id)
	// Ethereum uses V = 27 or 28
	return &HLSignature{
		R: "0x" + hex.EncodeToString(sig[:32]),
		S: "0x" + hex.EncodeToString(sig[32:64]),
		V: int(sig[64]) + 27,
	}, nil
}

// DeriveHLAddress derives the checksummed Ethereum address from a hex private key.
func DeriveHLAddress(hexKey string) (string, error) {
	privKey, err := hlParsePrivateKey(hexKey)
	if err != nil {
		return "", err
	}
	pubKey := privKey.Public().(*ecdsa.PublicKey)
	return crypto.PubkeyToAddress(*pubKey).Hex(), nil
}

// hlParsePrivateKey parses a hex-encoded ECDSA private key (with or without 0x prefix).
func hlParsePrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(hexKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("hlParsePrivateKey: %w", err)
	}
	return privKey, nil
}

// hlFloatToWire converts a float64 to Hyperliquid's 8-decimal wire format,
// removing trailing zeros (matching Python's Decimal.normalize()).
// E.g.: 95000.5 → "95000.5", 100.0 → "100"
func hlFloatToWire(x float64) string {
	s := fmt.Sprintf("%.8f", x)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// hlRoundSigFigs rounds x to n significant figures.
func hlRoundSigFigs(x float64, n int) float64 {
	if x == 0 {
		return 0
	}
	d := math.Ceil(math.Log10(math.Abs(x)))
	pow := math.Pow(10, float64(n)-d)
	return math.Round(x*pow) / pow
}

// hlRoundDecimalPlaces rounds x to n digits after the decimal point.
func hlRoundDecimalPlaces(x float64, decimalPlaces int) float64 {
	if decimalPlaces <= 0 {
		return math.Round(x)
	}
	pow := math.Pow(10, float64(decimalPlaces))
	return math.Round(x*pow) / pow
}

const hlPerpMaxDecimals = 6
const hlPerpIntegerPriceThreshold = 100_000

func hlPerpMaxPriceDecimals(szDecimals int) int {
	maxPriceDecimals := hlPerpMaxDecimals - szDecimals
	if maxPriceDecimals < 0 {
		return 0
	}
	return maxPriceDecimals
}

// hlFormatPerpPrice formats a perp price per Hyperliquid tick rules:
// https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size
// 5 significant figures, then max (6 - szDecimals) decimal places.
// Prices above 100k are rounded to integers since integer values are always accepted.
func hlFormatPerpPrice(px float64, szDecimals int) float64 {
	if px > hlPerpIntegerPriceThreshold {
		return math.Round(px)
	}

	sigFigs := hlRoundSigFigs(px, 5)
	maxPriceDecimals := hlPerpMaxPriceDecimals(szDecimals)

	return hlRoundDecimalPlaces(sigFigs, maxPriceDecimals)
}

// hlPriceToWire formats a perp price for the Hyperliquid wire API.
func hlPriceToWire(px float64, szDecimals int) string {
	return hlFloatToWire(hlFormatPerpPrice(px, szDecimals))
}
