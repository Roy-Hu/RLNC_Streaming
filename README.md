# 529_mp_stream

## TODO

- Enabling XNC in MPQUIC, please check 4.3 in CellFusion: Multipath Vehicle-to-Cloud Video Streaming with Network Coding in the Wild for implementation

## Setup

To run the server, please download the movie by get_your_movies.sh in goDASHbed (tos_4sec_full is enough, comment other folders)

Notice that is file is over 50G, make sure you have enough space on your VM

Save the files as /var/www/html/tos_4sec_full/4K_dataset/4_sec/x264/bbb/DASH_Files/full/<files> 

To run server:

    cd server
    
    go run server.go
    
To run godash in quic

    cd godash
    
    go build
    
    ./godash -config ./config/configure.json
