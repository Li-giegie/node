module example

go 1.21.3

require github.com/Li-giegie/node v1.0.0

require github.com/Li-giegie/go-utils v0.0.0-20231213033017-c0e274a2b5a0 // indirect

require (
<<<<<<< HEAD
	github.com/Li-giegie/go-jeans v1.0.1-0.20231213020253-59cc4b80f0d5 // indirect
=======
	github.com/Li-giegie/go-jeans v1.0.1-0.20240101191241-d4e67218fd82 // indirect
	github.com/Li-giegie/go-utils v0.0.0-20231213033017-c0e274a2b5a0 // indirect
>>>>>>> dev231223
	github.com/panjf2000/ants/v2 v2.9.0 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)

replace github.com/Li-giegie/node v1.0.0 => ../../node
