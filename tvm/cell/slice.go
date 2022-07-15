package cell

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
)

type Slice struct {
	special  bool
	level    byte
	bitsSz   uint
	loadedSz uint
	data     []byte

	// store it as slice of pointers to make indexing logic cleaner on parse,
	// from outside it should always come as object to not have problems
	refs []*Slice
}

func (c *Slice) MustLoadRef() *Slice {
	r, err := c.LoadRef()
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Slice) LoadRef() (*Slice, error) {
	if len(c.refs) == 0 {
		return nil, ErrNoMoreRefs
	}
	ref := c.refs[0]
	c.refs = c.refs[1:]

	return ref, nil
}

func (c *Slice) MustLoadMaybeRef() *Slice {
	r, err := c.LoadMaybeRef()
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Slice) LoadMaybeRef() (*Slice, error) {
	has, err := c.LoadBoolBit()
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, nil
	}

	if len(c.refs) == 0 {
		return nil, ErrNoMoreRefs
	}
	ref := c.refs[0]
	c.refs = c.refs[1:]

	return ref, nil
}

func (c *Slice) RefsNum() int {
	return len(c.refs)
}

func (c *Slice) MustLoadCoins() uint64 {
	r, err := c.LoadCoins()
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Slice) LoadCoins() (uint64, error) {
	value, err := c.LoadBigCoins()
	if err != nil {
		return 0, err
	}
	return value.Uint64(), nil
}

func (c *Slice) MustLoadBigCoins() *big.Int {
	r, err := c.LoadBigCoins()
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Slice) LoadBigCoins() (*big.Int, error) {
	// varInt 16 https://github.com/ton-blockchain/ton/blob/24dc184a2ea67f9c47042b4104bbb4d82289fac1/crypto/block/block-parse.cpp#L319
	ln, err := c.LoadUInt(4)
	if err != nil {
		return nil, err
	}

	value, err := c.LoadBigUInt(uint(ln) * 8)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (c *Slice) MustLoadUInt(sz uint) uint64 {
	res, err := c.LoadUInt(sz)
	if err != nil {
		panic(err)
	}
	return res
}

func (c *Slice) LoadUInt(sz uint) (uint64, error) {
	res, err := c.LoadBigUInt(sz)
	if err != nil {
		return 0, err
	}
	return res.Uint64(), nil
}

func (c *Slice) MustLoadInt(sz uint) int64 {
	res, err := c.LoadInt(sz)
	if err != nil {
		panic(err)
	}
	return res
}

func (c *Slice) LoadInt(sz uint) (int64, error) {
	res, err := c.LoadBigInt(sz)
	if err != nil {
		return 0, err
	}
	return res.Int64(), nil
}

func (c *Slice) MustLoadBoolBit() bool {
	r, err := c.LoadBoolBit()
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Slice) LoadBoolBit() (bool, error) {
	res, err := c.LoadBigUInt(1)
	if err != nil {
		return false, err
	}
	return res.Uint64() == 1, nil
}

func (c *Slice) MustLoadBigUInt(sz uint) *big.Int {
	r, err := c.LoadBigUInt(sz)
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Slice) LoadBigUInt(sz uint) (*big.Int, error) {
	if sz > 256 {
		return nil, ErrTooBigValue
	}

	b, err := c.LoadSlice(sz)
	if err != nil {
		return nil, err
	}

	// check is value is uses full bytes
	if offset := sz % 8; offset > 0 {
		// move bits to right side of bytes
		for i := len(b) - 1; i >= 0; i-- {
			b[i] >>= 8 - offset // get last bits
			if i > 0 {
				b[i] += b[i-1] << offset
			}
		}
	}

	return new(big.Int).SetBytes(b), nil
}

func (c *Slice) LoadBigInt(sz uint) (*big.Int, error) {
	u, err := c.LoadBigUInt(sz)
	if err != nil {
		return nil, err
	}

	one := big.NewInt(1)

	// check is last bit = 1
	isNegative := new(big.Int).And(u, new(big.Int).Lsh(one, uint(sz-1))).Cmp(big.NewInt(0)) != 0

	if isNegative {
		// get max value of given sz
		i := new(big.Int).Lsh(one, uint(sz))
		i = i.Sub(i, one)

		val := u.Sub(u, i)

		// finally, we are subtracting 1, because 0xFF = -1,
		// and when we do u-i we get value which is +1 from actual
		return val.Sub(val, one), nil
	}

	return u, nil
}

func (c *Slice) LoadVarUInt(sz uint) (*big.Int, error) {
	ln, err := c.LoadUInt(uint(big.NewInt(int64(sz - 1)).BitLen()))
	if err != nil {
		return nil, err
	}

	value, err := c.LoadBigUInt(uint(ln * 8))
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (c *Slice) MustLoadSlice(sz uint) []byte {
	s, err := c.LoadSlice(sz)
	if err != nil {
		panic(err)
	}
	return s
}

func (c *Slice) LoadSlice(sz uint) ([]byte, error) {
	if c.bitsSz-c.loadedSz < sz {
		return nil, ErrNotEnoughData
	}

	if sz <= 0 {
		return []byte{}, nil
	}

	leftSz := sz
	var unusedBits = uint(0)
	if l := c.loadedSz % 8; l > 0 && c.loadedSz > 0 {
		unusedBits = 8 - (c.loadedSz % 8)
	}

	var loadedData []byte

	var oneMoreLeft, oneMoreRight uint
	if unusedBits > 0 && sz > unusedBits {
		oneMoreLeft = 1
	}
	if (sz-unusedBits)%8 != 0 || sz-unusedBits == 0 {
		oneMoreRight = 1
	}

	ln := (sz-unusedBits)/8 + oneMoreLeft + oneMoreRight

	i := oneMoreLeft
	for leftSz > 0 {
		var b byte
		if oneMoreLeft > 0 {
			b = c.data[i-1] << byte(8-unusedBits)
			if i < ln {
				b += c.data[i] >> unusedBits
			}
		} else {
			b = c.data[i]
			if unusedBits > 0 {
				b <<= byte(8 - unusedBits)
			}
		}

		if leftSz < 8 {
			b &= 0xFF << (8 - leftSz)
			leftSz = 0
			loadedData = append(loadedData, b)
			break
		}

		if i < ln {
			loadedData = append(loadedData, b)
		}

		leftSz -= 8
		i++
	}

	if sz >= unusedBits {
		usedBytes := (sz - unusedBits) / 8
		if unusedBits > 0 {
			usedBytes++
		}

		c.data = c.data[usedBytes:]
	}

	c.loadedSz += sz

	return loadedData, nil
}

func (c *Slice) MustLoadAddr() *address.Address {
	a, err := c.LoadAddr()
	if err != nil {
		panic(err)
	}
	return a
}

func (c *Slice) LoadAddr() (*address.Address, error) {
	typ, err := c.LoadUInt(2)
	if err != nil {
		return nil, err
	}

	switch typ {
	case 2:
		isAnycast, err := c.LoadBoolBit()
		if err != nil {
			return nil, fmt.Errorf("failed to load anycast bit: %w", err)
		}

		if isAnycast {
			depthLen := uint(math.Ceil(math.Log2(30)))

			depth, err := c.LoadUInt(depthLen)
			if err != nil {
				return nil, fmt.Errorf("failed to load depth: %w", err)
			}

			pfx, err := c.LoadSlice(uint(depth))
			if err != nil {
				return nil, fmt.Errorf("failed to load prefix: %w", err)
			}
			_ = pfx

			// TODO: save anycast
		}

		workchain, err := c.LoadUInt(8)
		if err != nil {
			return nil, fmt.Errorf("failed to load workchain: %w", err)
		}

		data, err := c.LoadSlice(256)
		if err != nil {
			return nil, fmt.Errorf("failed to load addr data: %w", err)
		}

		return address.NewAddress(0, byte(workchain), data), nil
	case 0:
		return &address.Address{}, nil
	default:
		// TODO: support of all types of addresses, currently only std supported, skipping 3 bits
		return nil, errors.New("not supported type of address, currently only std supported")
	}

}

func (c *Slice) BitsLeft() uint {
	return c.bitsSz - c.loadedSz
}

func (c *Slice) RestBits() (uint, []byte, error) {
	left := c.bitsSz - c.loadedSz
	data, err := c.LoadSlice(left)
	return left, data, err
}

func (c *Slice) MustToCell() *Cell {
	cl, err := c.ToCell()
	if err != nil {
		panic(err)
	}
	return cl
}

func (c *Slice) Copy() *Slice {
	// copy data
	data := append([]byte{}, c.data...)

	var refs []*Slice
	for _, ref := range c.refs {
		refs = append(refs, ref.Copy())
	}

	return &Slice{
		bitsSz:   c.bitsSz,
		loadedSz: c.loadedSz,
		data:     data,
		refs:     refs,
	}
}

func (c *Slice) ToCell() (*Cell, error) {
	cp := c.Copy()

	left := cp.bitsSz - cp.loadedSz
	data, err := cp.LoadSlice(left)
	if err != nil {
		return nil, err
	}

	var refs []*Cell
	for _, ref := range cp.refs {
		cc, err := ref.ToCell()
		if err != nil {
			return nil, err
		}

		refs = append(refs, cc)
	}

	return &Cell{
		special: c.special,
		level:   c.level,
		bitsSz:  left,
		data:    data,
		refs:    refs,
	}, nil
}
