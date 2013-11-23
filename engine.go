package gapstone

// #cgo CFLAGS: -I/usr/include/capstone
// #cgo LDFLAGS: -lcapstone
// #include <stdlib.h>
// #include <capstone.h>
import "C"
import "unsafe"
import "reflect"
import "fmt"

type Arch uint
type Mode uint

type Engine struct {
	Handle C.csh
	Arch   Arch
	Mode   Mode
}

type InstructionHeader struct {
	Id               uint
	Address          uint
	Size             uint
	Mnemonic         string
	OpStr            string
	RegistersRead    []uint
	RegistersWritten []uint
	Groups           []uint
}

type Instruction struct {
	InstructionHeader
	Arm   ArmInstruction
	Arm64 Arm64Instruction
	Mips  MipsInstruction
	X86   X86Instruction
}

func fillGenericHeader(raw C.cs_insn, insn *Instruction) {
	insn.Id = uint(raw.id)
	insn.Address = uint(raw.address)
	insn.Size = uint(raw.size)
	insn.Mnemonic = C.GoString(&raw.mnemonic[0])
	insn.OpStr = C.GoString(&raw.op_str[0])
	for i := 0; raw.regs_read[i] != 0; i++ {
		insn.RegistersRead = append(insn.RegistersRead, uint(raw.regs_read[i]))
	}
	for i := 0; raw.regs_write[i] != 0; i++ {
		insn.RegistersWritten = append(insn.RegistersWritten, uint(raw.regs_write[i]))
	}
	for i := 0; raw.groups[i] != 0; i++ {
		insn.Groups = append(insn.Groups, uint(raw.groups[i]))
	}
}

func (e Engine) Close() (bool, error) {
	res, err := C.cs_close(e.Handle)
	return bool(res), err
}

func (e Engine) Version() (int, int) {
	var maj, min int
	C.cs_version((*C.int)(unsafe.Pointer(&maj)), (*C.int)(unsafe.Pointer(&min)))
	return maj, min
}

func (e Engine) RegName(reg uint) string {
	return C.GoString(C.cs_reg_name(e.Handle, C.uint(reg)))
}

func (e Engine) InsnName(insn uint) string {
	return C.GoString(C.cs_insn_name(e.Handle, C.uint(insn)))
}
func (e Engine) Disasm(input []byte, offset, count uint64) ([]Instruction, error) {

	var insn *C.cs_insn

	bptr := (*C.char)(unsafe.Pointer(&input[0]))
	disassembled := C.cs_disasm_dyn(
		e.Handle,
		bptr,
		C.uint64_t(len(input)),
		C.uint64_t(offset),
		C.uint64_t(count),
		&insn,
	)
	defer C.cs_free(unsafe.Pointer(insn))

	if disassembled > 0 {
		// Create a slice, and reflect its header
		var insns []C.cs_insn
		h := (*reflect.SliceHeader)(unsafe.Pointer(&insns))
		// Manually fill in the ptr, len and cap from the raw C data
		h.Data = uintptr(unsafe.Pointer(insn))
		h.Len = int(disassembled)
		h.Cap = int(disassembled)
		switch e.Arch {
		case CS_ARCH_ARM:
			return DecomposeArm(insns), nil
		case CS_ARCH_ARM64:
			return DecomposeArm64(insns), nil
		case CS_ARCH_MIPS:
			return DecomposeMips(insns), nil
		case CS_ARCH_X86:
			return DecomposeX86(insns), nil
		default:
			panic("Not implemented!")
		}
	}
	return nil, fmt.Errorf("Disassembly failed.")
}

func New(arch Arch, mode Mode) (Engine, error) {
	var handle C.csh
	res, err := C.cs_open(C.cs_arch(arch), C.cs_mode(mode), &handle)
	if res {
		return Engine{handle, arch, mode}, nil
	}
	return Engine{}, err
}
