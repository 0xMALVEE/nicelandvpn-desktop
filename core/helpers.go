package core

import (
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"runtime/debug"

	"golang.org/x/crypto/chacha20poly1305"
)

func RecoverAndLogToFile() {
	if r := recover(); r != nil {
		if LogFile != nil {
			CreateErrorLog("", r)
			CreateErrorLog("", string(debug.Stack()))
		} else {
			log.Println(r, string(debug.Stack()))
		}
	}
}

func CopySlice(in []byte) (out []byte) {
	out = make([]byte, len(in))
	_ = copy(out, in)
	return
}

func ReadMIDAndDataFromBuffer(CONN net.Conn, TunnelBuffer []byte) (n int, DL int, err error) {

	n, err = io.ReadAtLeast(CONN, TunnelBuffer[:MIDBufferLength], MIDBufferLength)
	if err != nil {
		CreateErrorLog("", "TUNNEL READER ERROR: ", err)
		return
	}

	if n < MIDBufferLength {
		CreateErrorLog("", "TUNNEL SMALL READ ERROR: ", CONN.RemoteAddr())
		err = errors.New("")
		return
	}

	DL = int(binary.BigEndian.Uint16(TunnelBuffer[6:8]))

	if DL > 0 {
		n, err = io.ReadAtLeast(CONN, TunnelBuffer[MIDBufferLength:MIDBufferLength+DL], DL)
		if err != nil {
			CreateErrorLog("", "TUNNEL DATA READ ERROR: ", err)
			return
		}
	}

	return
}

func GenerateEllipticCurveAndPrivateKey() (PK *ecdsa.PrivateKey, R *OTK_REQUEST, err error) {
	defer RecoverAndLogToFile()

	E := elliptic.P521()
	PK, err = ecdsa.GenerateKey(E, crand.Reader)
	if err != nil {
		CreateErrorLog("", "Unable to generate private key: ", err)
		return nil, nil, err
	}

	R = new(OTK_REQUEST)
	R.X = PK.PublicKey.X
	R.Y = PK.PublicKey.Y
	return
}

func GenerateAEADFromPrivateKey(PK *ecdsa.PrivateKey, R *OTK_REQUEST) (AEAD cipher.AEAD, err error) {
	var CCKeyb *big.Int
	var CCKey [32]byte
	defer func() {
		CCKeyb = nil
		CCKey = [32]byte{}
	}()
	defer RecoverAndLogToFile()

	CCKeyb, _ = PK.Curve.ScalarMult(R.X, R.Y, PK.D.Bytes())
	CCKey = sha256.Sum256(CCKeyb.Bytes())
	AEAD, err = chacha20poly1305.NewX(CCKey[:])
	if err != nil {
		CreateErrorLog("", "Unable to generate AEAD: ", err)
	}
	return
}

func SetGlobalStateAsDisconnected() {
	CreateLog("", "App state set to -Disconnected-")
	GLOBAL_STATE.Connected = false
	GLOBAL_STATE.Connecting = false
}