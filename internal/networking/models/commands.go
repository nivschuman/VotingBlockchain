package networking_models

var (
	CommandVersion   = [12]byte{'v', 'e', 'r', 's', 'i', 'o', 'n'}
	CommandVerAck    = [12]byte{'v', 'e', 'r', 'a', 'c', 'k'}
	CommandPing      = [12]byte{'p', 'i', 'n', 'g'}
	CommandPong      = [12]byte{'p', 'o', 'n', 'g'}
	CommandGetBlocks = [12]byte{'g', 'e', 't', 'b', 'l', 'o', 'c', 'k', 's'}
	CommandInv       = [12]byte{'i', 'n', 'v'}
	CommandGetData   = [12]byte{'g', 'e', 't', 'd', 'a', 't', 'a'}
	CommandBlock     = [12]byte{'b', 'l', 'o', 'c', 'k'}
	CommandTx        = [12]byte{'t', 'x'}
	CommandAlert     = [12]byte{'a', 'l', 'e', 'r', 't'}
	CommandReject    = [12]byte{'r', 'e', 'j', 'e', 'c', 't'}
	CommandAddr      = [12]byte{'a', 'd', 'd', 'r'}
	CommandGetAddr   = [12]byte{'g', 'e', 't', 'a', 'd', 'd', 'r'}
	CommandNotFound  = [12]byte{'n', 'o', 't', 'f', 'o', 'u', 'n', 'd'}
	CommandMemPool   = [12]byte{'m', 'e', 'm', 'p', 'o', 'o', 'l'}
)
