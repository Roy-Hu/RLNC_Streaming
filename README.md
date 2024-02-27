# 529_mp_stream

To run the server, please download the moive by get_your_movies.sh in goDASHbed (tos_4sec_full is enough, comment other folders)
Notice that is file is over 50G, make sure you have enough space on your VM

To run server:
    cd server
    go run server.go
To run godash
    cd godash
    go build
    ./godash -config ./config/configure.json