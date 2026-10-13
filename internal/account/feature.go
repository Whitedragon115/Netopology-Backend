// Package account: The signed-in user's own account (/me), account administration by the server owner/admins (/users) and server settings (/instance/settings).
//
// Operations (method, path under /api/v1, operationId, x-permission). Copy summaries, statuses
// and schemas from api/contract.yaml; follow internal/setup as the reference implementation.
//
//	WP-03  GET    /me                                                  getMe                    me.read
//	WP-04  PATCH  /me                                                  updateMe                 me.update
//	WP-04  POST   /me/password                                         changeMyPassword         me.password
//	WP-04  GET    /me/identities                                       listMyIdentities         me.identity_list
//	WP-04  DELETE /me/identities/{identityId}                          unlinkMyIdentity         me.identity_unlink
//	WP-04  GET    /users                                               listUsers                user.list
//	WP-04  POST   /users                                               createUser               user.create
//	WP-04  GET    /users/lookup                                        lookupUser               user.lookup
//	WP-04  GET    /users/{userId}                                      getUser                  user.read
//	WP-04  PATCH  /users/{userId}                                      updateUser               user.update
//	WP-04  POST   /users/{userId}/password                             resetUserPassword        user.password_reset
//	WP-04  GET    /instance/settings                                   getInstanceSettings      instance.settings_read
//	WP-04  PATCH  /instance/settings                                   updateInstanceSettings   instance.settings_update
//	WP-05  GET    /me/identities/{provider}/link                       linkMyIdentity           me.identity_link
package account
