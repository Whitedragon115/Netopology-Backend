// Package auth: Sign-in, sign-out and OAuth (Google, Discord). Also owns sessions (session.go) and the session middleware (middleware.go), which are implemented.
//
// Operations (method, path under /api/v1, operationId, x-permission). Copy summaries, statuses
// and schemas from api/contract.yaml; follow internal/setup as the reference implementation.
//
//	WP-03  POST   /auth/login                                          login                    auth.login
//	WP-03  POST   /auth/logout                                         logout                   auth.logout
//	WP-03  GET    /auth/providers                                      listAuthProviders        auth.providers
//	WP-05  GET    /auth/oauth/{provider}/start                         startOAuthSignIn         auth.oauth_start
//	WP-05  GET    /auth/oauth/{provider}/callback                      finishOAuthSignIn        auth.oauth_callback
package auth
