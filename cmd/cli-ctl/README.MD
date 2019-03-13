##Nested Installer

To build executables please run: `build.sh`

**config.yml** must be exists if not there is a __**config.sample.yml**__,
rename it and update the fields

**WARNING** if nested installer is not in /ronak/nested then you have to set an environment
variable NESTED_DIR and set it to point to nested directory.

### Available Commands:

1. **services**
    1. __install__          
        > Creates / updates docker compose files in yamls folder based on config.yml file parameters
    2. __update__            
        > Updates docker images
    3. __stop__              
        > Stops all running dockers
    4. __start__             
        > Starts all the installed dockers
2. **gen**
    1. __dkim__
        > Generates dkim public/private key pairs to be used in config.yml and name-server records
    2. __key__                     
        > Generates random key
        
        |Flags|Description|
        |:---:|:---:|
        | length | The length of key (default: 32) |
        
    3.  __selfSignedCertificate__   
        > Create self-signed certificate and its private key \
        
        |Flags|Description|
        |:---:|:---:|
        | --ou | Organization Unit (default: Nested Services) |
        | --cn | Common Name (default: *.nested.me) |
        | name | The file name of certificate (.crt) and private key (.key) |
        | singleFile | If true put certificate and key in a single file with .pem extention |
       
            
