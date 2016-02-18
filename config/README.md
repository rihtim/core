# Config

Config package is used for storing data provided with **dock-config.json** and used while booting the server. **dock-config.json** file must be at root directory of the project while running the Dock server.

**Example**

```
{
	"mongo": {
   		"address":  "192.168.59.103",
	    "name":     "dock-db"
	}
}
```

## Fields

### Mongo

An object contains the Mongo server's address and name.

**address**: IP address of the Mongo server. (required)

**name**: Name of the database on the Mongo server. (required)