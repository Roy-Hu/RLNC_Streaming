# 529_mp_stream

## Testing

Before testing, setup the rootDir of the testfile and the testFile name in xnc/config.go

To test, we need to introduce packet loss. 

First, use the command sudo tc qdisc add dev lo root netem loss 10% to set up a 10% packet loss rate. 

(Note: lo refers to the local network device. It might differ in a VirtualBox environment, so use ifconfig to verify.)

Next, uncomment the "loss debug" line in the xnc folder to enable the printing of debugging messages.

Then, execute go run example/filetransfer/filetransfer.go.

Ensure that some packets in each chunk are lost.

The file should be successfully decoded, as it will resend packets twice the amount needed.

## Setup

To run the server, please download the movie by get_your_movies.sh in goDASHbed (tos_4sec_full is enough, comment other folders)

Notice that is file is over 50G, make sure you have enough space on your VM

Save the DASH videos at /var/www/html/tos_4sec_full/4K_dataset/4_sec/x264/bbb/DASH_Files/full/<files> (Or change to the rootDir in xnc.Config)

To run server:

    cd server
    
    go run xnc/example/server.go
    
To run godash in quic

    cd godash
    
    go build
    
    ./godash -config ./config/configure.json
