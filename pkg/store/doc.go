// Repository interface package which contains the required methods for the
// auth and world server to store and retrive persistent data.
// If you need a specific store, you can implement your own. There are two
// implementations yet:
//   - localdb: this is a local YAML based store. (good for local development, or testing)
//   - mysqldb: MySQL implementation with Bun ORM (https://bun.uptrace.dev)
package store
