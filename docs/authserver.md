# 🏔 Auth Server

This is the authentication/realmlist provider component of the Summit WoW emulator.

The current architecture is the following:

![Authentication architecture](auth-server-arch.png)

### Compontents:
- Realmlist provider
- Accounts provider
- gRPC connector (functions for world server)

All components are pluggable, you can write your own implementation if you like to


# Running an auth server

There are different options:
- From binary distribution, downloadable from releases page
- Container - (kubernetes deployment/docs later 
  - [ ] #9
  - [ ] #10
- From code:
    <script src="https://gist.github.com/paalgyula/2cecca24d88c5bad94f2ccd4161a20ff.js"></script>

## Configuration

@paalgyula - todo