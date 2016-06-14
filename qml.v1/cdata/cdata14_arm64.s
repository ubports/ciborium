// +build go1.4

#include "textflag.h"

TEXT ·Ref(SB),NOSPLIT,$8-8
	BL runtime·acquirem(SB)
	MOVD 8(RSP), R0
	MOVD R0, ret+0(FP)
	MOVD R0, 8(RSP)
	BL runtime·releasem(SB)
	RET

TEXT ·Addrs(SB),NOSPLIT,$0-16
	MOVD	$runtime·main(SB), R0
	MOVD	R0, ret+0(FP)
	MOVD	$runtime·main_main(SB), R0
	MOVD	R0, ret+8(FP)
	RET
