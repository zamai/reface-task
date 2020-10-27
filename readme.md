To start server localy: 
1. `make pg` - start local pg container 
2. `make migrateup` - create db and apply migrations
3. `make server` - start dev server localy
4. `make kill_pg` - remove pg container 

Tests: 

`make test`

I decided to make test intergration level and run them with inside the tests with `dockertest`. 