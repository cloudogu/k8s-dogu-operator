docker rm -f samba; docker run --name samba -it --rm -p 445:445 -e "USER=admin" -e "PASS=admin" -v $PWD:/storage dockurr/samba
