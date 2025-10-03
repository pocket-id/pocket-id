## v1.12.0

### Bug Fixes

- do not use cache=shared for in-memory SQLite ([#971](https://github.com/pocket-id/pocket-id/pull/971) by @ItalyPaleAle)
- show only country in audit log location if no city instead of Unknown ([#977](https://github.com/pocket-id/pocket-id/pull/977) by @vilisseranen)
- display login location correctly if country or city is not present ([79989fb](https://github.com/pocket-id/pocket-id/commit/79989fb176273cef070dc52c338004b443364db8) by @stonith404)
- remove previous socket file to prevent bind error ([#979](https://github.com/pocket-id/pocket-id/pull/979) by @Caian)
- tokens issued with refresh token flow don't contain groups ([#989](https://github.com/pocket-id/pocket-id/pull/989) by @ItalyPaleAle)
- make logo and oidc client images sizes consistent ([01db8c0](https://github.com/pocket-id/pocket-id/commit/01db8c0a46b69a15a40951ba863e6bc08fa8e1f8) by @stonith404)
- include port in OIDC client details ([2c1c67b](https://github.com/pocket-id/pocket-id/commit/2c1c67b5e403b365204854c5eb222a68236f3ce0) by @stonith404)
- prevent endless effect loop in login wrapper ([fc9939d](https://github.com/pocket-id/pocket-id/commit/fc9939d1f1817c0b014cc54e6525b98762835295) by @stonith404)
- improve back button handling on auth pages ([d47b203](https://github.com/pocket-id/pocket-id/commit/d47b20326f96b6fff405fcc211719bf3068085ee) by @stonith404)
- allow any image source but disallow base64 ([22f4254](https://github.com/pocket-id/pocket-id/commit/22f42549323fde8b9eaeff682bfa4c7f27e05526) by @stonith404)
- date locale can't be loaded if locale is `en` ([b81de45](https://github.com/pocket-id/pocket-id/commit/b81de451668c425bfc5ca7cd6071fe2756b31594) by @stonith404)

### Features

- support for url based icons ([#840](https://github.com/pocket-id/pocket-id/pull/840) by @kmendell)
- hide alternative sign in methods page if email login disabled ([d010be4](https://github.com/pocket-id/pocket-id/commit/d010be4c8804153b4a7f55bd4ea1cedb0df471df) by @stonith404)
- add required indicator for required inputs ([#993](https://github.com/pocket-id/pocket-id/pull/993) by @stonith404)
- add the ability to make email optional ([#994](https://github.com/pocket-id/pocket-id/pull/994) by @stonith404)

### Other

- fix whitespace after commit message ([e8b172f](https://github.com/pocket-id/pocket-id/commit/e8b172f1c3df8eca8f463d7fa25a483b90a7e66c) by @stonith404)
- update AAGUIDs ([#972](https://github.com/pocket-id/pocket-id/pull/972) by @github-actions[bot])
- remove unnecessary logo fallback ([b746ac0](https://github.com/pocket-id/pocket-id/commit/b746ac0835da059e747a829df3a74e1eae79e107) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.11.2...v1.12.0

## v1.11.2

### Bug Fixes

- embedded paths not found on windows ([c55143d](https://github.com/pocket-id/pocket-id/commit/c55143d8c995fcd604edcdd448c50669e8682e33) by @stonith404)
- do not treat certain failures in app images bootstrap as fatal ([#966](https://github.com/pocket-id/pocket-id/pull/966) by @ItalyPaleAle)
- decouple images from app config service ([#965](https://github.com/pocket-id/pocket-id/pull/965) by @stonith404)

### Other

- use git cliff for release notes ([fde4e9b](https://github.com/pocket-id/pocket-id/commit/fde4e9b38a34331137a64ce328dad6faf9885808) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.11.1...v1.11.2

## v1.11.1

### Bug Fixes

- add missing translations([8c9cac2](https://github.com/pocket-id/pocket-id/commit/8c9cac2655ddbe4872234a1b55fdd51d2f3ac31c) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.11.0...v1.11.1

## v1.11.0

### Bug Fixes

- update localized name and description of ldap group name attribute ([#892](https://github.com/pocket-id/pocket-id/pull/892) by @kmendell)
- disable sign up options in UI if `UI_CONFIG_DISABLED` ([1d7cbc2](https://github.com/pocket-id/pocket-id/commit/1d7cbc2a4ecf352d46087f30b477f6bbaa23adf5) by @stonith404)
- ensure users imported from LDAP have fields validated ([#923](https://github.com/pocket-id/pocket-id/pull/923) by @ItalyPaleAle)
- list items on previous page get unselected if other items selected on next page ([6c696b4](https://github.com/pocket-id/pocket-id/commit/6c696b46c8b60b3dc4af35c9c6cf1b8e1322f4cd) by @stonith404)
- add validation for callback URLs ([#929](https://github.com/pocket-id/pocket-id/pull/929) by @stonith404)
- key-rotate doesn't work with database storage ([#940](https://github.com/pocket-id/pocket-id/pull/940) by @ItalyPaleAle)
- make environment variables case insensitive where necessary ([#954](https://github.com/pocket-id/pocket-id/pull/954) by @stonith404)
- my apps card shouldn't take full width if only one item exists ([e7e53a8](https://github.com/pocket-id/pocket-id/commit/e7e53a8b8c87bee922167d24556aef3ea219b1a2) by @stonith404)

### Features

- add custom base url ([#858](https://github.com/pocket-id/pocket-id/pull/858) by @DerSteph)
- client_credentials flow support ([#901](https://github.com/pocket-id/pocket-id/pull/901) by @savely-krasovsky)
- add info box to app settings if UI config is disabled ([a1d8538](https://github.com/pocket-id/pocket-id/commit/a1d8538c64beb4d7e8559934985772fba27623ca) by @stonith404)
- add CSP header ([#908](https://github.com/pocket-id/pocket-id/pull/908) by @stonith404)
- return new id_token when using refresh token ([#925](https://github.com/pocket-id/pocket-id/pull/925) by @ItalyPaleAle)
- add PWA support ([#938](https://github.com/pocket-id/pocket-id/pull/938) by @stonith404)
- add support for `LOG_LEVEL` env variable ([#942](https://github.com/pocket-id/pocket-id/pull/942) by @stonith404)
- add user display name field ([#898](https://github.com/pocket-id/pocket-id/pull/898) by @kmendell)
- allow uppercase usernames ([#958](https://github.com/pocket-id/pocket-id/pull/958) by @stonith404)

### Other

- use react email for email templates ([#734](https://github.com/pocket-id/pocket-id/pull/734) by @kmendell)
- update AAGUIDs ([#903](https://github.com/pocket-id/pocket-id/pull/903) by @github-actions[bot])
- add Swedish files ([954fb4f](https://github.com/pocket-id/pocket-id/commit/954fb4f0c8c3126738baa30431e32bad6afaa9f5) by @kmendell)
- update AAGUIDs ([#926](https://github.com/pocket-id/pocket-id/pull/926) by @github-actions[bot])
- bump vite from 7.0.6 to 7.0.7 in the npm_and_yarn group across 1 directory ([#932](https://github.com/pocket-id/pocket-id/pull/932) by @dependabot[bot])
- bump axios from 1.11.0 to 1.12.0 in the npm_and_yarn group across 1 directory ([#943](https://github.com/pocket-id/pocket-id/pull/943) by @dependabot[bot])
- minify background image ([#933](https://github.com/pocket-id/pocket-id/pull/933) by @ItalyPaleAle)
- include version in changelog ([cf08929](https://github.com/pocket-id/pocket-id/commit/cf0892922beb56552504025cbfb710878caf7de4) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.10.0...v1.11.0

## v1.10.0

### Bug Fixes

- apps showed multiple times if user is in multiple groups ([641bbc9](https://github.com/pocket-id/pocket-id/commit/641bbc935191bad8afbfec90943fc3e9de7a0cb6) by @stonith404)

### Features

- redesigned sidebar with administrative dropdown ([#881](https://github.com/pocket-id/pocket-id/pull/881) by @kmendell)

### Other

- update AAGUIDs ([#885](https://github.com/pocket-id/pocket-id/pull/885) by @github-actions[bot])
- bump sveltekit to 2.36.3 and devalue to 5.3.2 ([#889](https://github.com/pocket-id/pocket-id/pull/889) by @kmendell)
- add missing translations ([#884](https://github.com/pocket-id/pocket-id/pull/884) by @savely-krasovsky)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.9.1...v1.10.0

## v1.9.1

### Bug Fixes

- sqlite migration drops allowed user groups ([d6d1a4c](https://github.com/pocket-id/pocket-id/commit/d6d1a4ced23886f255a9c2048d19ad3599a17f26) by @stonith404)

### Other

- add no tx wrap to unit tests ([51222f5](https://github.com/pocket-id/pocket-id/commit/51222f5607a172c67028d821ec2648be53e5776c) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.9.0...v1.9.1

## v1.9.0

### Bug Fixes

- don't force uuid for client id in postgres ([2ffc6ba](https://github.com/pocket-id/pocket-id/commit/2ffc6ba42af4742a13b77543142b66b3e826ab88) by @stonith404)
- sort order incorrect for apps when using postgres ([d0392d2](https://github.com/pocket-id/pocket-id/commit/d0392d25edcaa5f3c7da2aad70febf63b47763fa) by @stonith404)
- ensure SQLite has a writable temporary directory ([#876](https://github.com/pocket-id/pocket-id/pull/876) by @ItalyPaleAle)

### Features

- support automatic db migration rollbacks ([#874](https://github.com/pocket-id/pocket-id/pull/874) by @stonith404)

### Other

- add postgres down migration to 20250822000000 ([63db4d5](https://github.com/pocket-id/pocket-id/commit/63db4d51208af62bf960a5b4ce88674281ecb01d) by @stonith404)
- fix postgres e2e tests ([#877](https://github.com/pocket-id/pocket-id/pull/877) by @stonith404)
- fix playwright browsers not installed ([8999173](https://github.com/pocket-id/pocket-id/commit/8999173aa00e43ea6edac38c5637f4cbaf032c32) by @stonith404)
- use TEXT instead of VARCHAR for client ID ([654593b](https://github.com/pocket-id/pocket-id/commit/654593b4b602c9b3d9d45e500fb5c088ad58b2ee) by @stonith404)
- use matrix for e2e tests ([c1e515a](https://github.com/pocket-id/pocket-id/commit/c1e515a05fe584b17f7f1485a598a32e168d83a8) by @stonith404)
- bump golang.org/x/oauth2 from 0.26.0 to 0.27.0 in /backend in the go_modules group across 1 directory ([#879](https://github.com/pocket-id/pocket-id/pull/879) by @dependabot[bot])

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.8.1...v1.9.0

## v1.8.1

### Bug Fixes

- wrong column type for reauthentication tokens in Postgres ([#869](https://github.com/pocket-id/pocket-id/pull/869) by @ItalyPaleAle)
- migration clears allowed users groups ([5971bfb](https://github.com/pocket-id/pocket-id/commit/5971bfbfa66ecfebf2b1c08d34fcbd8c18cdc046) by @stonith404)

### Other

- update issue template ([#870](https://github.com/pocket-id/pocket-id/pull/870) by @ItalyPaleAle)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.8.0...v1.8.1

## v1.8.0

### Bug Fixes

- non admin users can't revoke oidc client but see edit link ([0e44f24](https://github.com/pocket-id/pocket-id/commit/0e44f245afcdf8179bf619613ca9ef4bffa176ca) by @stonith404)
- ignore client secret if client is public ([#836](https://github.com/pocket-id/pocket-id/pull/836) by @James18232)
- bump rollup from 4.45.3 to 4.46.3 ([#845](https://github.com/pocket-id/pocket-id/pull/845) by @gepbird)
- delete webauthn session after login to prevent replay attacks ([fe003b9](https://github.com/pocket-id/pocket-id/commit/fe003b927ce7772692439992860c804de89ce424) by @stonith404)
- move audit log call before TX is committed ([#854](https://github.com/pocket-id/pocket-id/pull/854) by @ItalyPaleAle)
- for one-time access tokens and signup tokens, pass TTLs instead of absolute expiration date ([#855](https://github.com/pocket-id/pocket-id/pull/855) by @ItalyPaleAle)
- authorization can't be revoked ([0aab3f3](https://github.com/pocket-id/pocket-id/commit/0aab3f3c7ad8c1b14939de3ded60c9f201eab8fc) by @stonith404)
- ferated identities can't be cleared ([24e2742](https://github.com/pocket-id/pocket-id/commit/24e274200fe4002d01c58cc3fa74094b598d7599) by @stonith404)
- oidc client advanced options color ([fc0c99a](https://github.com/pocket-id/pocket-id/commit/fc0c99a232b0efb1a5b5d2c551102418b1080293) by @stonith404)
- enable foreign key check for sqlite ([#863](https://github.com/pocket-id/pocket-id/pull/863) by @stonith404)

### Features

- display all accessible oidc clients in the dashboard ([#832](https://github.com/pocket-id/pocket-id/pull/832) by @stonith404)
- login code font change ([#851](https://github.com/pocket-id/pocket-id/pull/851) by @James18232)
- add option to OIDC client to require re-authentication ([#747](https://github.com/pocket-id/pocket-id/pull/747) by @MorrisMorrison)
- add default user groups and claims for new users ([#812](https://github.com/pocket-id/pocket-id/pull/812) by @zeedif)
- allow custom client IDs ([#864](https://github.com/pocket-id/pocket-id/pull/864) by @stonith404)

### Other

- update AAGUIDs ([#826](https://github.com/pocket-id/pocket-id/pull/826) by @github-actions[bot])
- update deps and Go 1.25 ([#833](https://github.com/pocket-id/pocket-id/pull/833) by @ItalyPaleAle)
- update AAGUIDs ([#844](https://github.com/pocket-id/pocket-id/pull/844) by @github-actions[bot])
- add Korean files ([d77d8eb](https://github.com/pocket-id/pocket-id/commit/d77d8eb0680061ef141e16f36b30ab679553a3ea) by @kmendell)
- use proper async calls for cleanupBackend function ([#846](https://github.com/pocket-id/pocket-id/pull/846) by @kmendell)
- strip debug symbol from backend binary ([#856](https://github.com/pocket-id/pocket-id/pull/856) by @maximerobine)
- change alternative sign in methods text ([c51265d](https://github.com/pocket-id/pocket-id/commit/c51265dafb4c0344456dc55fa5a395dfb0e1f5ca) by @stonith404)
- run formatter ([2c122d4](https://github.com/pocket-id/pocket-id/commit/2c122d413d07e4dd8c8551676ad9d8dc433328ee) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.7.0...v1.8.0

## v1.7.0

### Bug Fixes

- set input type 'email' for email-based login ([#776](https://github.com/pocket-id/pocket-id/pull/776) by @ItalyPaleAle)
- delete WebAuthn registration session after use ([#783](https://github.com/pocket-id/pocket-id/pull/783) by @ItalyPaleAle)
- admins can not delete or disable their own account ([f0c144c](https://github.com/pocket-id/pocket-id/commit/f0c144c51c635bc348222a00d3bc88bc4e0711ef) by @kmendell)
- authorization animation not working ([9ac5d51](https://github.com/pocket-id/pocket-id/commit/9ac5d5118710cad59c8c4ce7cef7ab09be3de664) by @stonith404)
- custom claims input suggestions instantly close after opening ([4d59e72](https://github.com/pocket-id/pocket-id/commit/4d59e7286666480e20c728787a95e82513509240) by @stonith404)

### Features

- Support OTel and JSON for logs (via log/slog) ([#760](https://github.com/pocket-id/pocket-id/pull/760) by @ItalyPaleAle)
- add support for `code_challenge_methods_supported` ([#794](https://github.com/pocket-id/pocket-id/pull/794) by @kmendell)
- support reading secret env vars from \_FILE ([#799](https://github.com/pocket-id/pocket-id/pull/799) by @ItalyPaleAle)
- add robots.txt to block indexing ([#806](https://github.com/pocket-id/pocket-id/pull/806) by @Etienne-bdt)
- user application dashboard ([#727](https://github.com/pocket-id/pocket-id/pull/727) by @kmendell)

### Other

- add Ukrainian files ([51b73c9](https://github.com/pocket-id/pocket-id/commit/51b73c9c3162f956ac8bf5de54fad03ec6c18bb2) by @kmendell)
- bump form-data from 4.0.1 to 4.0.4 in /frontend in the npm_and_yarn group across 1 directory ([#771](https://github.com/pocket-id/pocket-id/pull/771) by @dependabot[bot])
- bump axios from 1.10.0 to 1.11.0 in /frontend in the npm_and_yarn group across 1 directory ([#777](https://github.com/pocket-id/pocket-id/pull/777) by @dependabot[bot])
- add Vietnamese files ([60f0b28](https://github.com/pocket-id/pocket-id/commit/60f0b280767ace788409de87a7e9d7928f200bf0) by @kmendell)
- rename glass-row-item to passkey-row ([c359b5b](https://github.com/pocket-id/pocket-id/commit/c359b5be065887b7526463adf9f349b9d586b75c) by @kmendell)
- update dependencies and fix zod/4 import path ([ffed465](https://github.com/pocket-id/pocket-id/commit/ffed465f09e174c969fce23674c68ecbd482a1ce) by @kmendell)
- update dependencies and fix zod/4 import path ([f3c6521](https://github.com/pocket-id/pocket-id/commit/f3c6521f2be633b107eaf0d1839db37b17d60638) by @kmendell)
- fix federated credentials type error ([56ee7d9](https://github.com/pocket-id/pocket-id/commit/56ee7d946fc3cad1ed8a94f3665d393de4d8b81e) by @kmendell)
- update Vietnamese display name ([12a7a6a](https://github.com/pocket-id/pocket-id/commit/12a7a6a5c5bec4f209260b90fc06e2e2201aecdf) by @kmendell)
- complete conversion of log calls to slog ([#787](https://github.com/pocket-id/pocket-id/pull/787) by @ItalyPaleAle)
- additional logs for database connections ([#813](https://github.com/pocket-id/pocket-id/pull/813) by @ItalyPaleAle)
- use reflection to mark file based env variables ([#815](https://github.com/pocket-id/pocket-id/pull/815) by @stonith404)
- switch from npm to pnpm ([#786](https://github.com/pocket-id/pocket-id/pull/786) by @kmendell)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.6.4...v1.7.0

## v1.6.4

### Bug Fixes

- migration fails on postgres ([#762](https://github.com/pocket-id/pocket-id/pull/762) by @ItalyPaleAle)

### Other

- remove labels from issue templates ([4c76de4](https://github.com/pocket-id/pocket-id/commit/4c76de45ed24ae899dfa35d96a87fec2d22a2a2a) by @kmendell)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.6.3...v1.6.4

## v1.6.3

### Bug Fixes

- allow passkey names up to 50 characters ([b03e91b](https://github.com/pocket-id/pocket-id/commit/b03e91b6530c2393ad20ac49aa2cb2b4962651b2) by @kmendell)
- use object-contain for images on oidc-client list ([d3bc179](https://github.com/pocket-id/pocket-id/commit/d3bc1797b65ec8bc9201c55d06f3612093f3a873) by @kmendell)
- ensure user inputs are normalized ([#724](https://github.com/pocket-id/pocket-id/pull/724) by @ItalyPaleAle)
- use user-agent for identifying known device signins ([ef1d599](https://github.com/pocket-id/pocket-id/commit/ef1d5996624fc534190f80a26f2c48bbad206f49) by @kmendell)
- show rename and delete buttons for passkeys without hovering over the row ([2952b15](https://github.com/pocket-id/pocket-id/commit/2952b1575542ecd0062fe740e2d6a3caad05190d) by @kmendell)

### Other

- use issue types for new issues ([db94f81](https://github.com/pocket-id/pocket-id/commit/db94f8193784d2f02b588aa7d2295716f00eea80) by @kmendell)
- use correct svelte 5 syntax for signup token modal ([f145903](https://github.com/pocket-id/pocket-id/commit/f145903eb09a5b22647694bf8483559197e1663c) by @kmendell)
- upgrade dependencies ([#752](https://github.com/pocket-id/pocket-id/pull/752) by @kmendell)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.6.2...v1.6.3

## v1.6.2

### Bug Fixes

- login failures on Postgres when IP is null ([#737](https://github.com/pocket-id/pocket-id/pull/737) by @ItalyPaleAle)
- ensure confirmation dialog shows on top of other components ([f103a54](https://github.com/pocket-id/pocket-id/commit/f103a547904070c5b192e519c8b5a8fed9d80e96) by @kmendell)

### Other

- update AAGUIDs ([#729](https://github.com/pocket-id/pocket-id/pull/729) by @github-actions[bot])
- Fix inconsistent punctuation marks for the language name of zh-TW ([#731](https://github.com/pocket-id/pocket-id/pull/731) by @xlionjuan)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.6.1...v1.6.2

## v1.6.1

### Other

- use `latest-distroless` tag for latest distroless images ([f565c70](https://github.com/pocket-id/pocket-id/commit/f565c702e57c390e079de258fcd46239af26d96e) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.6.0...v1.6.1

## v1.6.0

### Bug Fixes

- add missing error check in initial user setup ([fceb6fa](https://github.com/pocket-id/pocket-id/commit/fceb6fa7b4701a3645c4c2353bcd108b15d69ded) by @stonith404)
- app config forms not updating with latest values ([#696](https://github.com/pocket-id/pocket-id/pull/696) by @kmendell)
- auth fails when client IP is empty on Postgres ([#695](https://github.com/pocket-id/pocket-id/pull/695) by @ItalyPaleAle)
- token introspection authentication not handled correctly ([#704](https://github.com/pocket-id/pocket-id/pull/704) by @stonith404)
- allow profile picture update even if "allow own account edit" enabled ([9872608](https://github.com/pocket-id/pocket-id/commit/9872608d61a486f7b775f314d9392e0620bcd891) by @stonith404)
- support non UTF-8 LDAP IDs ([#714](https://github.com/pocket-id/pocket-id/pull/714) by @stonith404)
- linter issues ([#719](https://github.com/pocket-id/pocket-id/pull/719) by @ItalyPaleAle)
- actually fix linter issues ([#720](https://github.com/pocket-id/pocket-id/pull/720) by @ItalyPaleAle)
- show friendly name in user group selection ([5c9e504](https://github.com/pocket-id/pocket-id/commit/5c9e504291b3bffe947bcbe907701806e301d1fe) by @stonith404)
- keep sidebar in settings sticky ([e46f60a](https://github.com/pocket-id/pocket-id/commit/e46f60ac8d6944bcea54d0708af1950d98f66c3c) by @stonith404)
- custom claims input suggestions flickering ([49f1ab2](https://github.com/pocket-id/pocket-id/commit/49f1ab2f75df97d551fff5acbadcd55df74af617) by @stonith404)

### Features

- enhance language selection message and add translation contribution link ([be52660](https://github.com/pocket-id/pocket-id/commit/be526602273c1689cb4057ca96d4214e7f817d1d) by @stonith404)
- encrypt private keys saved on disk and in database ([#682](https://github.com/pocket-id/pocket-id/pull/682) by @ItalyPaleAle)
- add "key-rotate" command ([#709](https://github.com/pocket-id/pocket-id/pull/709) by @ItalyPaleAle)
- distroless container additional variant + healthcheck command ([#716](https://github.com/pocket-id/pocket-id/pull/716) by @ItalyPaleAle)
- add support for OAuth 2.0 Authorization Server Issuer Identification ([bf04256](https://github.com/pocket-id/pocket-id/commit/bf042563e997d57bb087705a5789fd72ffbed467) by @stonith404)

### Other

- use github.com/jinzhu/copier for MapStruct ([#698](https://github.com/pocket-id/pocket-id/pull/698) by @ItalyPaleAle)
- add CODEOWNERS file ([2ecc1ab](https://github.com/pocket-id/pocket-id/commit/2ecc1abbad5899823fdcda60d0df0c773cd1bb2e) by @kmendell)
- update CODEOWNERS to be global ([459a4fd](https://github.com/pocket-id/pocket-id/commit/459a4fd727e9b64f2c4eb9da2d1c096aac8c4fec) by @kmendell)
- use correct team name for codeowners ([3a29027](https://github.com/pocket-id/pocket-id/commit/3a2902789efa0c22f60e3b04c49c1a2ca131a5f9) by @kmendell)
- run formatter ([857b9cc](https://github.com/pocket-id/pocket-id/commit/857b9cc864fd77ed93346c46cf1f97b896e112f2) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.5.0...v1.6.0

## v1.5.0

### Bug Fixes

- remove duplicate request logging ([#678](https://github.com/pocket-id/pocket-id/pull/678) by @ryankask)
- error page flickering after sign out ([1a77bd9](https://github.com/pocket-id/pocket-id/commit/1a77bd9914ea01e445ff3d6e116c9ed3bcfbf153) by @stonith404)
- users can't be updated by admin if self account editing is disabled ([29cb551](https://github.com/pocket-id/pocket-id/commit/29cb5513a03d1a9571969c8a42deec9b2bdee037) by @stonith404)
- less noisy logging for certain GET requests ([#681](https://github.com/pocket-id/pocket-id/pull/681) by @11notes)
- margin of user sign up description ([052ac00](https://github.com/pocket-id/pocket-id/commit/052ac008c3a8c910d1ce79ee99b2b2f75e4090f4) by @stonith404)
- improve accent color picker disabled state ([d976bf5](https://github.com/pocket-id/pocket-id/commit/d976bf5965eda10e3ecb71821c23e93e5d712a02) by @stonith404)
- double double full stops for certain error messages ([d070b9a](https://github.com/pocket-id/pocket-id/commit/d070b9a778d7d1a51f2fa62d003f2331a96d6c91) by @stonith404)

### Documentation

- clarify confusing user update logic ([1fdb058](https://github.com/pocket-id/pocket-id/commit/1fdb058386a175107c5d28eb2e59eab1954756ad) by @stonith404)

### Features

- self-service user signup ([#672](https://github.com/pocket-id/pocket-id/pull/672) by @kmendell)
- redact sensitive app config variables if set with env variable ([ba61cdb](https://github.com/pocket-id/pocket-id/commit/ba61cdba4eb3d5659f3ae6b6c21249985c0aa630) by @stonith404)
- improve initial admin creation workflow ([287314f](https://github.com/pocket-id/pocket-id/commit/287314f01644e42ddb2ce1b1115bd14f2f0c1768) by @stonith404)

### Other

- add formatter to Playwright tests ([73e7e0b](https://github.com/pocket-id/pocket-id/commit/73e7e0b1c548f322e6a646cc43ec069cc04132c2) by @stonith404)
- fix e2e tests ([4b82975](https://github.com/pocket-id/pocket-id/commit/4b829757b24149f56a57d3e6574018c162367843) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.4.1...v1.5.0

## v1.4.1

### Bug Fixes

- app not starting if UI config is disabled and Postgres is used ([7d36bda](https://github.com/pocket-id/pocket-id/commit/7d36bda769e25497dec6b76206a4f7e151b0bd72) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.4.0...v1.4.1

## v1.4.0

### Bug Fixes

- allow images with uppercase file extension ([1bcb50e](https://github.com/pocket-id/pocket-id/commit/1bcb50edc335886dd722a4c69960c48cc3cd1687) by @stonith404)
- reduce duration of animations on login and signin page ([#648](https://github.com/pocket-id/pocket-id/pull/648) by @ItalyPaleAle)
- center oidc client images if they are smaller than the box ([946c534](https://github.com/pocket-id/pocket-id/commit/946c534b0877a074a6b658060f9af27e4061397c) by @stonith404)
- explicitly cache images to prevent unexpected behavior ([2e5d268](https://github.com/pocket-id/pocket-id/commit/2e5d2687982186c12e530492292d49895cb6043a) by @stonith404)
- use inline style for dynamic background image URL instead of Tailwind class ([bef77ac](https://github.com/pocket-id/pocket-id/commit/bef77ac8dca2b98b6732677aaafbc28f79d00487) by @stonith404)

### Features

- auto-focus on the login buttons ([#647](https://github.com/pocket-id/pocket-id/pull/647) by @ItalyPaleAle)
- use icon instead of text on application image update hover state ([215531d](https://github.com/pocket-id/pocket-id/commit/215531d65c6683609b0b4a5505fdb72696fdb93e) by @stonith404)
- ui accent colors ([#643](https://github.com/pocket-id/pocket-id/pull/643) by @kmendell)
- allow setting unix socket mode ([#661](https://github.com/pocket-id/pocket-id/pull/661) by @CnTeng)
- location filter for global audit log ([#662](https://github.com/pocket-id/pocket-id/pull/662) by @kmendell)
- configurable local ipv6 ranges for audit log ([#657](https://github.com/pocket-id/pocket-id/pull/657) by @kmendell)

### Other

- Update spelling and grammar in en.json ([#650](https://github.com/pocket-id/pocket-id/pull/650) by @amazingca)
- run formatter ([fd3c76f](https://github.com/pocket-id/pocket-id/commit/fd3c76ffa37969b13fea43a625dc2ea5a7027692) by @stonith404)
- run formatter ([5814549](https://github.com/pocket-id/pocket-id/commit/5814549cbe80e926c9f1b9abf6ee9f3315e84154) by @stonith404)
- only build required binaries for `next` image ([3717a66](https://github.com/pocket-id/pocket-id/commit/3717a663d96331b6b266d91c57abc2ba94513ce9) by @stonith404)
- cancel `build-next` action if new one starts ([c77167d](https://github.com/pocket-id/pocket-id/commit/c77167df462bed512c7d3275aade6e2c5e7d180d) by @stonith404)
- use `v1` tag in example `docker-compose.yml` ([c8eb034](https://github.com/pocket-id/pocket-id/commit/c8eb034c492001deb3a47dd3f3c2a366f784bb3c) by @stonith404)
- remove unused crypto util ([d5928f6](https://github.com/pocket-id/pocket-id/commit/d5928f6fea0268b8c64e7b5dba5218208f269c37) by @stonith404)
- add configuration for backend hot reloading ([481df3b](https://github.com/pocket-id/pocket-id/commit/481df3bcb9816df27feba71853f02214e9b9809c) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.3.1...v1.4.0

## v1.3.1

### Bug Fixes

- change timestamp of `client_credentials.sql` migration ([2935236](https://github.com/pocket-id/pocket-id/commit/2935236acee9c78c2fe6787ec8b5f53ae0eca047) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.3.0...v1.3.1

## v1.3.0

### Bug Fixes

- don't load app config and user on every route change ([bdcef60](https://github.com/pocket-id/pocket-id/commit/bdcef60cab6a61e1717661e918c42e3650d23fee) by @stonith404)
- UI config overridden by env variables don't apply on first start ([5e9096e](https://github.com/pocket-id/pocket-id/commit/5e9096e328741ba2a0e03835927fe62e6aea2a89) by @stonith404)
- OIDC client image can't be deleted ([61b62d4](https://github.com/pocket-id/pocket-id/commit/61b62d461200c1359a16c92c9c62530362a4785c) by @stonith404)
- use full width for audit log filters ([575b2f7](https://github.com/pocket-id/pocket-id/commit/575b2f71e9f1ff9c4f6fd411b136676c213b7201) by @stonith404)
- misleading text for disable animations option ([657a51f](https://github.com/pocket-id/pocket-id/commit/657a51f7ed8a77e8a937971032091058aacfded6) by @stonith404)

### Documentation

- fix pagination API docs ([ea4e486](https://github.com/pocket-id/pocket-id/commit/ea4e48680c12f433900246240f56b440d9bbea4a) by @stonith404)
- remove difficult to maintain OpenAPI properties ([3cc82d8](https://github.com/pocket-id/pocket-id/commit/3cc82d8522b2e1107a312d9ff89683af99af76fd) by @stonith404)

### Features

- add API endpoint for user authorized clients ([d217083](https://github.com/pocket-id/pocket-id/commit/d217083059120171d5c555b09eefe6ba3c8a8d42) by @stonith404)
- add unix socket support ([#615](https://github.com/pocket-id/pocket-id/pull/615) by @CnTeng)
- JWT bearer assertions for client authentication ([#566](https://github.com/pocket-id/pocket-id/pull/566) by @ItalyPaleAle)
- oidc client data preview ([#624](https://github.com/pocket-id/pocket-id/pull/624) by @kmendell)
- new color theme for the UI ([97f7326](https://github.com/pocket-id/pocket-id/commit/97f7326da40265a954340d519661969530f097a0) by @stonith404)
- allow introspection and device code endpoints to use Federated Client Credentials ([#640](https://github.com/pocket-id/pocket-id/pull/640) by @ItalyPaleAle)

### Other

- run fomratter ([dc5d7bb](https://github.com/pocket-id/pocket-id/commit/dc5d7bb2f3eae4759760c8243562fae0a56be374) by @stonith404)
- add Danish language files ([b650d6d](https://github.com/pocket-id/pocket-id/commit/b650d6d423ce5ee4676bb0c214b1f23cc0913cde) by @stonith404)
- add Traditional Chinese files ([31a803b](https://github.com/pocket-id/pocket-id/commit/31a803b2430ee1926a857f586a698ac54c29f9f6) by @stonith404)
- add workflow for building 'next' docker image ([#633](https://github.com/pocket-id/pocket-id/pull/633) by @kmendell)
- upgrade to Zod v4 ([#623](https://github.com/pocket-id/pocket-id/pull/623) by @stonith404)
- add missing permission ([f403eed](https://github.com/pocket-id/pocket-id/commit/f403eed12cec0ea6f548d9fa79d20727bf06cf64) by @stonith404)
- add missing attestions permission ([b25e95f](https://github.com/pocket-id/pocket-id/commit/b25e95fc4aed239f0d85771c28fe2cbacdc94112) by @stonith404)
- update AAGUIDs ([#639](https://github.com/pocket-id/pocket-id/pull/639) by @github-actions[bot])
- upgrade to Shadcn v1.0.0 ([242d87a](https://github.com/pocket-id/pocket-id/commit/242d87a54bb9e85434e58349b029bbe5d10d9deb) by @stonith404)
- add docs link and rename to Federated Client Credentials ([#636](https://github.com/pocket-id/pocket-id/pull/636) by @ItalyPaleAle)
- add branch check to release script ([a09d529](https://github.com/pocket-id/pocket-id/commit/a09d529027472ce4f77211dbe0c090795a558dab) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.2.0...v1.3.0

## v1.2.0

### Bug Fixes

- show LAN for auditlog location for internal networks ([b874681](https://github.com/pocket-id/pocket-id/commit/b8746818240fde052e6f3b5db5c3355d7bbfcbda) by @kmendell)
- small fixes in analytics_job ([#582](https://github.com/pocket-id/pocket-id/pull/582) by @ItalyPaleAle)
- run jobs at interval instead of specific time ([#585](https://github.com/pocket-id/pocket-id/pull/585) by @ItalyPaleAle)
- don't use TOFU for logout callback URLs ([#588](https://github.com/pocket-id/pocket-id/pull/588) by @ItalyPaleAle)
- clear default app config variables from database ([decf8ec](https://github.com/pocket-id/pocket-id/commit/decf8ec70b5f6a69fe201d6e4ad60ee62e374ad0) by @stonith404)
- allow users to update their locale even when own account update disabled ([6c00aaa](https://github.com/pocket-id/pocket-id/commit/6c00aaa3efa75c76d340718698a0f4556e8de268) by @stonith404)
- fallback to primary language if no translation available for specific country ([2440379](https://github.com/pocket-id/pocket-id/commit/2440379cd11b4a6da7c52b122ba8f49d7c72ce1d) by @stonith404)
- whitelist authorization header for CORS ([b9489b5](https://github.com/pocket-id/pocket-id/commit/b9489b5e9a32a2a3f54d48705e731a7bcf188d20) by @stonith404)
- improve spacing on auth screens ([04fcf11](https://github.com/pocket-id/pocket-id/commit/04fcf1110e97b42dc5f0c20e169c569075d1e797) by @stonith404)
- page scrolls up on form submisssion ([31ad904](https://github.com/pocket-id/pocket-id/commit/31ad904367e53dd47a15abcce5402dfe84828a14) by @stonith404)

### Documentation

- use https in `.env.example` ([c24a554](https://github.com/pocket-id/pocket-id/commit/c24a5546a5254d56f58658a2d3d74b5431508b67) by @stonith404)

### Features

- auto detect callback url ([#583](https://github.com/pocket-id/pocket-id/pull/583) by @kmendell)

### Other

- adapt unit test for new app config default value behavior ([00259f8](https://github.com/pocket-id/pocket-id/commit/00259f88195e56dcb3e0c2bb3c53a2f2c489d382) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.1.0...v1.2.0

## v1.1.0

### Bug Fixes

- use ldapAttributeUserUsername for finding group members ([#565](https://github.com/pocket-id/pocket-id/pull/565) by @kmendell)
- run user group count inside a transaction ([f03b80f](https://github.com/pocket-id/pocket-id/commit/f03b80f9d7f2529d8cef23ca6a742a914a4ec883) by @stonith404)

### Features

- require user verification for passkey sign in ([68e4b67](https://github.com/pocket-id/pocket-id/commit/68e4b67bd212e31ecc20277bfd293c94bf7f3642) by @stonith404)
- show allowed group count on oidc client list ([#567](https://github.com/pocket-id/pocket-id/pull/567) by @kmendell)
- add daily heartbeat request for counting Pocket ID instances ([#578](https://github.com/pocket-id/pocket-id/pull/578) by @stonith404)

### Other

- update AAGUIDs ([#576](https://github.com/pocket-id/pocket-id/pull/576) by @github-actions[bot])
- tag container images with v{major} ([#577](https://github.com/pocket-id/pocket-id/pull/577) by @maximbaz)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v1.0.0...v1.1.0

## v1.0.0

### Bug Fixes

- custom logo not correctly loaded if UI configuration is disabled ([bf710ae](https://github.com/pocket-id/pocket-id/commit/bf710aec5625c9dcb43c83d920318a036a135bae) by @stonith404)
- animation speed set to max of 300ms ([c726c16](https://github.com/pocket-id/pocket-id/commit/c726c1621b8bd88b20cb05263f6d10888f0af8e2) by @kmendell)
- authorize page doesn't load ([c3a03db](https://github.com/pocket-id/pocket-id/commit/c3a03db8b0f87cddc927481cfad2ccc391f98869) by @stonith404)
- ldap tests ([4dc0b2f](https://github.com/pocket-id/pocket-id/commit/4dc0b2f37f9a57ba1c7ea084dc2a713f283d1b14) by @kmendell)
- remove curly bracket from user group URL ([5fa15f6](https://github.com/pocket-id/pocket-id/commit/5fa15f60984a8f2a02f15900860c3a3097032e1b) by @stonith404)
- remove nested button in user group list ([f57c8d3](https://github.com/pocket-id/pocket-id/commit/f57c8d347e127027378aad8831a8e4dfebfef060) by @stonith404)
- add back month and year selection for date picker ([6c35570](https://github.com/pocket-id/pocket-id/commit/6c35570e78813ca6af1bae6a0374d7483bff9824) by @stonith404)
- show correct app name on sign out page ([131f470](https://github.com/pocket-id/pocket-id/commit/131f470757044fddd0989a76e9dc9e310f19819c) by @stonith404)
- use pointer cursor for menu items ([f820fc8](https://github.com/pocket-id/pocket-id/commit/f820fc830161499edb0da2df334e4e473d5825ae) by @stonith404)
- use same color as title for description in alert ([e19b33f](https://github.com/pocket-id/pocket-id/commit/e19b33fc2e2b9dd149da1f9351aca2e839ffae04) by @stonith404)
- trim whitespaces from string inputs ([#537](https://github.com/pocket-id/pocket-id/pull/537) by @stonith404)

### Documentation

- adapt contribution guide ([cbe7aa6](https://github.com/pocket-id/pocket-id/commit/cbe7aa6eecf0ba73cfe7f05db90ce63d893826ec) by @stonith404)

### Features

- improve buttons styling ([c37386f](https://github.com/pocket-id/pocket-id/commit/c37386f8b2f2c64bd9e7c437879a2217846852b5) by @stonith404)

### Other

- update AAGUIDs ([#523](https://github.com/pocket-id/pocket-id/pull/523) by @github-actions[bot])
- remove old DB env variables, and jwk migrations logic ([#529](https://github.com/pocket-id/pocket-id/pull/529) by @kmendell)
- switch SQLite driver to pure-Go implementation ([#530](https://github.com/pocket-id/pocket-id/pull/530) by @ItalyPaleAle)
- flaky unit test in db_bootstrap_test ([#532](https://github.com/pocket-id/pocket-id/pull/532) by @ItalyPaleAle)
- update options API for simplewebauthn ([#543](https://github.com/pocket-id/pocket-id/pull/543) by @RealOrangeOne)
- update AAGUIDs ([#547](https://github.com/pocket-id/pocket-id/pull/547) by @github-actions[bot])
- add Polish translations ([#554](https://github.com/pocket-id/pocket-id/pull/554) by @mikolaj92)
- serve the static frontend trough the backend ([f8a7467](https://github.com/pocket-id/pocket-id/commit/f8a7467ec0e939f90d19211a0a0efc5e17a58127) by @stonith404)
- update release pipelines ([35b227c](https://github.com/pocket-id/pocket-id/commit/35b227cd17efa1ff37d76aee09d2f0081a69df68) by @ItalyPaleAle)
- replace create-one-time-access-token script with in-app functionality ([cb2a9f9](https://github.com/pocket-id/pocket-id/commit/cb2a9f9f7d2cd7cd19ecabbcb883ac6c8118c4aa) by @ItalyPaleAle)
- address linter's complaint in 1.0 branch ([3896b7b](https://github.com/pocket-id/pocket-id/commit/3896b7bb3b30ff8887a30d95778829533d62ed40) by @ItalyPaleAle)
- some clean-up in OIDC service and controller ([b71c84c](https://github.com/pocket-id/pocket-id/commit/b71c84c355c8feaadd7799cc54ca444ef8abce43) by @ItalyPaleAle)
- remove pocket-id binary ([b2e8993](https://github.com/pocket-id/pocket-id/commit/b2e89934de61d5ec4df85a9d5186bb0be6f48321) by @kmendell)
- add pocket-id to .gitignore ([8326bfd](https://github.com/pocket-id/pocket-id/commit/8326bfd13694cf712f918510d200c7aba84d65b8) by @kmendell)
- add `.well-known` to development reverse proxy ([05b443d](https://github.com/pocket-id/pocket-id/commit/05b443d984ec7758fa85720c89c2671652ac3328) by @stonith404)
- migrate shadcn-components to Svelte 5 and TW4 ([28c8599](https://github.com/pocket-id/pocket-id/commit/28c85990baa473f6083660f21cc20ddffa58104c) by @kmendell)
- adapt e2e tests ([ac6df53](https://github.com/pocket-id/pocket-id/commit/ac6df536ef266d58eecbbfd7c077c56f959b12ec) by @stonith404)
- use bits-10 as selector ([21cb331](https://github.com/pocket-id/pocket-id/commit/21cb3310d66dcc6e3ad372d8873ba6e5629f3159) by @kmendell)
- wait for network ([53f212f](https://github.com/pocket-id/pocket-id/commit/53f212fd3a22b10e8221ed8647eb26453ab8f5f9) by @kmendell)
- move e2e tests to root of repository ([966a566](https://github.com/pocket-id/pocket-id/commit/966a566adeb5e128c0988239d06dbfb820cdce30) by @stonith404)
- start test containers with Docker Compose ([ebcf861](https://github.com/pocket-id/pocket-id/commit/ebcf861aa682c46294d88584e60cc004e229b4e3) by @stonith404)
- fix `.auth` path of e2e tests ([ca5e754](https://github.com/pocket-id/pocket-id/commit/ca5e754aea0a851eeb3cc044d6618c340653c189) by @stonith404)
- move `auth.setup.ts` into `specs` folder ([9fff6ec](https://github.com/pocket-id/pocket-id/commit/9fff6ec3b61ef6bb90e8b1aef51b8e95b7d617d5) by @stonith404)
- fix change locale test ([5b3ff7b](https://github.com/pocket-id/pocket-id/commit/5b3ff7b8798d7b2e241d3f996a1e4e37b85c0e09) by @stonith404)
- fix lldap setup if data already seeded ([3042de2](https://github.com/pocket-id/pocket-id/commit/3042de2ce1e9e3102eea464aab4ca2e93c5aea05) by @stonith404)
- add missing types to Playwright tests ([a65c0b3](https://github.com/pocket-id/pocket-id/commit/a65c0b3da346c5882fb4f5fee59edad32d6b5dba) by @stonith404)
- fix e2e tests after shadcn upgrade ([869c4c5](https://github.com/pocket-id/pocket-id/commit/869c4c5871b9b33044713fa98d79786b28b2939f) by @stonith404)
- exclude binary from project root ([0d4d538](https://github.com/pocket-id/pocket-id/commit/0d4d5386c77eb146064dcdc4feaa69709e27fbf5) by @stonith404)
- remove unused `data.json` ([2a457ac](https://github.com/pocket-id/pocket-id/commit/2a457ac8e9bb7917a703a948501fc1edff4ad27c) by @stonith404)
- add major flag to release script ([2793eb4](https://github.com/pocket-id/pocket-id/commit/2793eb4ebd50feb209e9e9f5f8516a3f0a15323c) by @stonith404)
- upgrade `build-push-action` ([ed0e566](https://github.com/pocket-id/pocket-id/commit/ed0e566e99c9a66623fc7a09eaa8d9472764ecb5) by @stonith404)
- remove default value from `TARGETARCH` in Dockerfile ([7691622](https://github.com/pocket-id/pocket-id/commit/769162227419bf379ebe39b285de76b7223c527a) by @stonith404)
- fix subject digest in container image attestation ([31ae8ca](https://github.com/pocket-id/pocket-id/commit/31ae8cac964668eeb361bc20eceba8e2eca3623e) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.53.0...v1.0.0

## v0.53.0

### Bug Fixes

- handle CORS correctly for endpoints that SPAs need ([#513](https://github.com/pocket-id/pocket-id/pull/513) by @stonith404)

### Features

- add support for `TZ` environment variable ([5e2e947](https://github.com/pocket-id/pocket-id/commit/5e2e947fe09fa881a7bbc70133a243a4baf30e90) by @stonith404)

### Other

- organize imports ([ba256c7](https://github.com/pocket-id/pocket-id/commit/ba256c76bc84d4acb904fcdf41728d8c9732cc48) by @stonith404)
- add e2e LDAP tests ([#466](https://github.com/pocket-id/pocket-id/pull/466) by @kmendell)
- remove wait for LDAP sync ([de648dd](https://github.com/pocket-id/pocket-id/commit/de648dd6daac8af51bed4fba695cc3c0e4a79039) by @stonith404)
- run formatter ([e0db469](https://github.com/pocket-id/pocket-id/commit/e0db4695acd82246bc638745d56e935b199f98b6) by @stonith404)
- add explicit permissions to actions ([90bdd29](https://github.com/pocket-id/pocket-id/commit/90bdd29fb67e9ffc13384b9b8ce19b76b789efc2) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.52.0...v0.53.0

## v0.52.0

### Bug Fixes

- correctly set script permissions inside Docker container ([c55fef0](https://github.com/pocket-id/pocket-id/commit/c55fef057cdcec867af91b29968541983cd80ec0) by @stonith404)

### Features

- OpenTelemetry tracing and metrics ([#495](https://github.com/pocket-id/pocket-id/pull/495) by @daenney)
- add healthz endpoint ([#494](https://github.com/pocket-id/pocket-id/pull/494) by @ItalyPaleAle)

### Other

- add svelte-check workflow for the frontend ([8ec2388](https://github.com/pocket-id/pocket-id/commit/8ec238826903f1daf557e8118b42c5b794c833a0) by @kmendell)
- build frontend to include paraglide before running svelte-check ([5d78445](https://github.com/pocket-id/pocket-id/commit/5d784455014adf29f51a5b7a48b7f9f673427308) by @kmendell)
- create a PR instead of commiting for update aaguids workflow ([364f5b3](https://github.com/pocket-id/pocket-id/commit/364f5b38b944b7c528fadb6ee0932cd0ac8d98b4) by @stonith404)
- update AAGUIDs ([#507](https://github.com/pocket-id/pocket-id/pull/507) by @github-actions[bot])

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.51.1...v0.52.0

## v0.51.1

### Bug Fixes

- last name still showing as required on account form ([#492](https://github.com/pocket-id/pocket-id/pull/492) by @kmendell)
- non admin users weren't able to call the end session endpoint ([6bd6cef](https://github.com/pocket-id/pocket-id/commit/6bd6cefaa6dc571a319a6a1c2b2facc2404eadd3) by @stonith404)
- allow LDAP users to update their locale ([0b9cbf4](https://github.com/pocket-id/pocket-id/commit/0b9cbf47e36a332cfd854aa92e761264fb3e4795) by @stonith404)

### Other

- bump vite from 6.2.6 to 6.3.4 in /frontend in the npm_and_yarn group across 1 directory ([#496](https://github.com/pocket-id/pocket-id/pull/496) by @dependabot[bot])
- complete graceful shutdown implementation and add service runner ([#493](https://github.com/pocket-id/pocket-id/pull/493) by @ItalyPaleAle)
- fix type errors ([f4c6cff](https://github.com/pocket-id/pocket-id/commit/f4c6cff4613ff986b3954403f5dd894c5fbf2dac) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.51.0...v0.51.1

## v0.51.0

### Bug Fixes

- updating scopes of an authorized client fails with Postgres ([0a24ab8](https://github.com/pocket-id/pocket-id/commit/0a24ab80010eb5a15d99915802c6698274a5c57c) by @stonith404)
- hide global audit log switch for non admin users ([1efd1d1](https://github.com/pocket-id/pocket-id/commit/1efd1d182dbb6190d3c7e27034426c9e48781b4a) by @stonith404)
- return correct error message if user isn't authorized ([86d2b5f](https://github.com/pocket-id/pocket-id/commit/86d2b5f59f26cb944017826cbd8df915cdc986f1) by @stonith404)
- do not require PKCE for public clients ([ce24372](https://github.com/pocket-id/pocket-id/commit/ce24372c571cc3b277095dc6a4107663d64f45b3) by @stonith404)

### Features

- new login code card position for mobile devices ([#452](https://github.com/pocket-id/pocket-id/pull/452) by @James18232)

### Other

- reorganize imports ([4614769](https://github.com/pocket-id/pocket-id/commit/4614769b84e6dfd9414eeeb2b347d056069beca2) by @stonith404)
- graceful shutdown for server ([#482](https://github.com/pocket-id/pocket-id/pull/482) by @ItalyPaleAle)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.50.0...v0.51.0

## v0.50.0

### Bug Fixes

- rootless Caddy data and configuration ([#470](https://github.com/pocket-id/pocket-id/pull/470) by @eiqnepm)
- do not override XDG_DATA_HOME/XDG_CONFIG_HOME if they are already set ([#472](https://github.com/pocket-id/pocket-id/pull/472) by @ItalyPaleAle)
- prevent deadlock when trying to delete LDAP users ([#471](https://github.com/pocket-id/pocket-id/pull/471) by @ItalyPaleAle)
- pass context to methods that were missing it ([#487](https://github.com/pocket-id/pocket-id/pull/487) by @ItalyPaleAle)

### Features

- make family name optional ([#476](https://github.com/pocket-id/pocket-id/pull/476) by @kmendell)
- device authorization endpoint ([#270](https://github.com/pocket-id/pocket-id/pull/270) by @kmendell)

### Other

- Add Simplified Chinese translation. ([#473](https://github.com/pocket-id/pocket-id/pull/473) by @Star-caorui)
- do not force redirects to happen on the server ([#481](https://github.com/pocket-id/pocket-id/pull/481) by @ItalyPaleAle)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.49.0...v0.50.0

## v0.49.0

### Bug Fixes

- locale change in dropdown doesn't work on first try ([60bad9e](https://github.com/pocket-id/pocket-id/commit/60bad9e9859d81c9967e6939e1ed10a65145a936) by @stonith404)
- remove limit of 20 callback URLs ([c37a3e0](https://github.com/pocket-id/pocket-id/commit/c37a3e0ed177c3bd2b9a618d1f4b0709004478b0) by @stonith404)
- disable animations not respected on authorize and logout page ([e571996](https://github.com/pocket-id/pocket-id/commit/e571996cb57d04232c1f47ab337ad656f48bb3cb) by @stonith404)
- hide alternative sign in button if user is already authenticated ([4e05b82](https://github.com/pocket-id/pocket-id/commit/4e05b82f02740a4bae07cec6c6a64acd34ca0fc3) by @stonith404)

### Features

- add description to callback URL inputs ([eb689eb](https://github.com/pocket-id/pocket-id/commit/eb689eb56ec9eaf8b0fb1485040e26f841b9225d) by @stonith404)
- send email to user when api key expires within 7 days ([#451](https://github.com/pocket-id/pocket-id/pull/451) by @kmendell)
- add ability to send login code via email ([#457](https://github.com/pocket-id/pocket-id/pull/457) by @stonith404)
- add ability to disable API key expiration email ([9122e75](https://github.com/pocket-id/pocket-id/commit/9122e75101ad39a40135ccf931eb2bfd351b5db6) by @stonith404)

### Other

- add kmendell to `FUNDING.yml` ([e21ee8a](https://github.com/pocket-id/pocket-id/commit/e21ee8a871134863e0834c32a58df3578dbd8289) by @stonith404)
- setup caching and improve ci job performance ([#465](https://github.com/pocket-id/pocket-id/pull/465) by @kmendell)
- fix type errors ([2597907](https://github.com/pocket-id/pocket-id/commit/2597907578ab4adffd32bf708580b1309c69d917) by @stonith404)
- fix typo in key ([55273d6](https://github.com/pocket-id/pocket-id/commit/55273d68c93a2470bba95b0b06ce430e17c0ddeb) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.48.0...v0.49.0

## v0.48.0

### Bug Fixes

- profile picture empty for users without first or last name ([#449](https://github.com/pocket-id/pocket-id/pull/449) by @kmendell)
- add "type" as reserved claim ([0111a58](https://github.com/pocket-id/pocket-id/commit/0111a58dac0342c5ac2fa25a050e8773810d2b0a) by @stonith404)
- callback URL doesn't get rejected if it starts with a different string ([f0dce41](https://github.com/pocket-id/pocket-id/commit/f0dce41fbc5649b3a8fe65de36ca20efa521b880) by @stonith404)
- user querying fails on global audit log page with Postgres ([84f1d5c](https://github.com/pocket-id/pocket-id/commit/84f1d5c906ec3f9a74ad3d2f36526eea847af5dd) by @stonith404)

### Features

- add gif support for logo and background image ([56a8b5d](https://github.com/pocket-id/pocket-id/commit/56a8b5d0c02643f869b77cf8475ddf2f9473880b) by @stonith404)
- disable/enable users ([#437](https://github.com/pocket-id/pocket-id/pull/437) by @kmendell)

### Other

- bump golang.org/x/net from 0.36.0 to 0.38.0 in /backend in the go_modules group across 1 directory ([#450](https://github.com/pocket-id/pocket-id/pull/450) by @dependabot[bot])
- add Italian ([75fbfee](https://github.com/pocket-id/pocket-id/commit/75fbfee4d8963fd3ab97d29063f527297e093069) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.47.0...v0.48.0

## v0.47.0

### Bug Fixes

- define token type as claim for better client compatibility ([adf7458](https://github.com/pocket-id/pocket-id/commit/adf74586afb6ef9a00fb122c150b0248c5bc23f0) by @stonith404)

### Features

- add qrcode representation of one time link ([#436](https://github.com/pocket-id/pocket-id/pull/436) by @paulgreg)
- disable animations setting toggle ([#442](https://github.com/pocket-id/pocket-id/pull/442) by @kmendell)

### Other

- bump @sveltejs/kit from 2.16.1 to 2.20.6 in /frontend in the npm_and_yarn group across 1 directory ([#443](https://github.com/pocket-id/pocket-id/pull/443) by @dependabot[bot])
- adapt JWTs in e2e tests ([9b2d622](https://github.com/pocket-id/pocket-id/commit/9b2d622990b3f81ff5bce64043c6fe1a1e4b6f69) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.46.0...v0.47.0

## v0.46.0

### Bug Fixes

- create reusable default profile pictures ([#406](https://github.com/pocket-id/pocket-id/pull/406) by @kmendell)
- ensure file descriptors are closed + other bugs ([#413](https://github.com/pocket-id/pocket-id/pull/413) by @ItalyPaleAle)
- ensure indexes on audit_logs table ([#415](https://github.com/pocket-id/pocket-id/pull/415) by @ItalyPaleAle)
- use transactions when operations involve multiple database queries ([#392](https://github.com/pocket-id/pocket-id/pull/392) by @ItalyPaleAle)
- use UUID for temporary file names ([ccc18d7](https://github.com/pocket-id/pocket-id/commit/ccc18d716f16a7ef1775d30982e2ba7b5ff159a6) by @stonith404)
- add missing rollback for LDAP sync ([658a9ca](https://github.com/pocket-id/pocket-id/commit/658a9ca6dd8d2304ff3639a000bab02e91ff68a6) by @stonith404)
- ignore profile picture cache after profile picture gets updated ([4ba6893](https://github.com/pocket-id/pocket-id/commit/4ba68938dd2a631c633fcb65d8c35cb039d3f59c) by @stonith404)
- improve LDAP error handling ([#425](https://github.com/pocket-id/pocket-id/pull/425) by @ItalyPaleAle)

### Documentation

- update swagger description to use markdown ([#418](https://github.com/pocket-id/pocket-id/pull/418) by @kmendell)

### Features

- modernize ui ([#381](https://github.com/pocket-id/pocket-id/pull/381) by @kmendell)
- global audit log ([#320](https://github.com/pocket-id/pocket-id/pull/320) by @kmendell)
- implement token introspection ([#405](https://github.com/pocket-id/pocket-id/pull/405) by @aksdb)
- Added button when you don't have a passkey added. ([#426](https://github.com/pocket-id/pocket-id/pull/426) by @arne)

### Other

- remove cors exception from middleware as this is handled by the handler ([cf3084c](https://github.com/pocket-id/pocket-id/commit/cf3084cfa8e151c78f55a24af05dddb9d3a0fc71) by @stonith404)
- improve czech translation strings ([#408](https://github.com/pocket-id/pocket-id/pull/408) by @jose-d)
- bump vite from 6.2.3 to 6.2.4 in /frontend in the npm_and_yarn group across 1 directory ([#410](https://github.com/pocket-id/pocket-id/pull/410) by @dependabot[bot])
- fix mistakes in source strings ([4627f36](https://github.com/pocket-id/pocket-id/commit/4627f365a2d7b227350087bf7f0e9c6dfde095f6) by @stonith404)
- bump vite from 6.2.4 to 6.2.5 in /frontend in the npm_and_yarn group across 1 directory ([#417](https://github.com/pocket-id/pocket-id/pull/417) by @dependabot[bot])
- rollback db changes with `defer` everywhere ([ce6e27d](https://github.com/pocket-id/pocket-id/commit/ce6e27d0ff3682c62f740e3c9103f515b3f16e9b) by @stonith404)
- simplify app_config service and fix race conditions ([#423](https://github.com/pocket-id/pocket-id/pull/423) by @ItalyPaleAle)
- bump vite from 6.2.5 to 6.2.6 in /frontend in the npm_and_yarn group across 1 directory ([#433](https://github.com/pocket-id/pocket-id/pull/433) by @dependabot[bot])

### Performance Improvements

- run async operations in parallel in server load functions ([1762629](https://github.com/pocket-id/pocket-id/commit/17626295964244c5582806bd0f413da2c799d5ad) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.45.0...v0.46.0

## v0.45.0

### Bug Fixes

- use value receiver for `AuditLogData` ([cbd1bbd](https://github.com/pocket-id/pocket-id/commit/cbd1bbdf741eedd03e93598d67623c75c74b6212) by @stonith404)
- ldap users aren't deleted if removed from ldap server ([7e65827](https://github.com/pocket-id/pocket-id/commit/7e658276f04d08a1f5117796e55d45e310204dab) by @stonith404)
- use WAL for SQLite by default and set busy_timeout ([#388](https://github.com/pocket-id/pocket-id/pull/388) by @ItalyPaleAle)

### Documentation

- update .env.example to reflect the new documentation location ([#385](https://github.com/pocket-id/pocket-id/pull/385) by @PsychotherapistSam)

### Features

- add support for ECDSA and EdDSA keys ([#359](https://github.com/pocket-id/pocket-id/pull/359) by @ItalyPaleAle)

### Other

- add basic static analysis for backend ([#389](https://github.com/pocket-id/pocket-id/pull/389) by @Rich7690)
- run linter only on backend changes ([6fa26c9](https://github.com/pocket-id/pocket-id/commit/6fa26c97be76140b58c78742ba97e1ac336c3ecb) by @stonith404)
- fix code smells ([c9e0073](https://github.com/pocket-id/pocket-id/commit/c9e0073b6362dffc93b79f340289853ba28aa9d6) by @stonith404)
- fix code smells ([5c198c2](https://github.com/pocket-id/pocket-id/commit/5c198c280cea4c5bf8572f34df5c84f4069d6b27) by @stonith404)
- migrate backend linter to v2. fixed unit test workflow ([#400](https://github.com/pocket-id/pocket-id/pull/400) by @Rich7690)
- install inlang plugins from npm ([#401](https://github.com/pocket-id/pocket-id/pull/401) by @gepbird)
- add swagger title and version info ([#399](https://github.com/pocket-id/pocket-id/pull/399) by @kmendell)
- add Brazilian Portuguese ([fc68cf7](https://github.com/pocket-id/pocket-id/commit/fc68cf7eb218689ce392226fca42b125fbe714e5) by @stonith404)
- do not include test controller in production builds ([#402](https://github.com/pocket-id/pocket-id/pull/402) by @ItalyPaleAle)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.44.0...v0.45.0

## v0.44.0

### Bug Fixes

- skip ldap objects without a valid unique id ([#376](https://github.com/pocket-id/pocket-id/pull/376) by @kmendell)
- hash the refresh token in the DB (security) ([#379](https://github.com/pocket-id/pocket-id/pull/379) by @ItalyPaleAle)
- stop container if Caddy, the frontend or the backend fails ([e6f5019](https://github.com/pocket-id/pocket-id/commit/e6f50191cf05a5d0ac0e0000cf66423646f1920e) by @stonith404)

### Documentation

- fix api routers for swag documentation ([#378](https://github.com/pocket-id/pocket-id/pull/378) by @kmendell)

### Features

- add OIDC refresh_token support ([#325](https://github.com/pocket-id/pocket-id/pull/325) by @kmendell)

### Other

- fix invalid action configuration ([edf1097](https://github.com/pocket-id/pocket-id/commit/edf1097dd3183adb25863ee0636913cf92c67877) by @stonith404)
- skip e2e tests if the PR comes from `i18n_crowdin` ([af5b2f7](https://github.com/pocket-id/pocket-id/commit/af5b2f7913480520c4d6702356a730568f44e606) by @stonith404)
- add Russian localization ([#371](https://github.com/pocket-id/pocket-id/pull/371) by @savely-krasovsky)
- bump github.com/golang-jwt/jwt/v5 from 5.2.1 to 5.2.2 in /backend in the go_modules group across 1 directory ([#374](https://github.com/pocket-id/pocket-id/pull/374) by @dependabot[bot])
- add French, Czech and German to language picker ([35766af](https://github.com/pocket-id/pocket-id/commit/35766af0556ba4bb5360c1680a892faa8b0bd3bc) by @stonith404)
- use atomic renames for uploaded files ([#372](https://github.com/pocket-id/pocket-id/pull/372) by @ItalyPaleAle)
- bump vite from 6.2.1 to 6.2.3 in /frontend in the npm_and_yarn group across 1 directory ([#384](https://github.com/pocket-id/pocket-id/pull/384) by @dependabot[bot])

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.43.1...v0.44.0

## v0.43.1

### Bug Fixes

- wrong base locale causes crash ([3120ebf](https://github.com/pocket-id/pocket-id/commit/3120ebf239b90f0bc0a0af33f30622e034782398) by @stonith404)

### Other

- ignore e2e tests on Crowdin branch ([2fb4193](https://github.com/pocket-id/pocket-id/commit/2fb41937cacd9173b95c251b7bf00850fd097ca7) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.43.0...v0.43.1

## v0.43.0

### Features

- name new passkeys based on agguids ([#332](https://github.com/pocket-id/pocket-id/pull/332) by @kmendell)
- add support for translations ([#349](https://github.com/pocket-id/pocket-id/pull/349) by @jonasclaes)

### Other

- update Crowdin configuration ([3ee26a2](https://github.com/pocket-id/pocket-id/commit/3ee26a2cfb4f2fc6b35b10ab75bb26bd94b789d7) by @stonith404)
- use language code with country for messages ([31ac560](https://github.com/pocket-id/pocket-id/commit/31ac56004ad7c7b52dbf7126da5d8d7f67b78b36) by @stonith404)
- remove unused messages ([bb23194](https://github.com/pocket-id/pocket-id/commit/bb23194e8858dda2c9e5570879a33547954606e9) by @stonith404)
- add language request issue template ([c578bab](https://github.com/pocket-id/pocket-id/commit/c578baba9507c4ca5919430442f9525adc64ca0a) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.42.1...v0.43.0

## v0.42.1

### Bug Fixes

- kid not added to JWTs ([f7e36a4](https://github.com/pocket-id/pocket-id/commit/f7e36a422ea6b5327360c9a13308ae408ff7fffe) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.42.0...v0.42.1

## v0.42.0

### Features

- store keys as JWK on disk ([#339](https://github.com/pocket-id/pocket-id/pull/339) by @ItalyPaleAle)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.41.0...v0.42.0

## v0.41.0

### Bug Fixes

- own avatar not loading ([#351](https://github.com/pocket-id/pocket-id/pull/351) by @savely-krasovsky)

### Features

- allow reset of profile picture ([#355](https://github.com/pocket-id/pocket-id/pull/355) by @kmendell)

### Other

- correct misspellings ([#352](https://github.com/pocket-id/pocket-id/pull/352) by @szepeviktor)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.40.1...v0.41.0

## v0.40.1

### Bug Fixes

- email logo icon displaying too big ([#336](https://github.com/pocket-id/pocket-id/pull/336) by @kmendell)
- Fixes and performance improvements in utils package ([#331](https://github.com/pocket-id/pocket-id/pull/331) by @ItalyPaleAle)
- remove custom claim key restrictions ([9f28503](https://github.com/pocket-id/pocket-id/commit/9f28503d6c73d3521d1309bee055704a0507e9b5) by @stonith404)
- API keys not working if sqlite is used ([8ead0be](https://github.com/pocket-id/pocket-id/commit/8ead0be8cd0cfb542fe488b7251cfd5274975ae1) by @stonith404)
- caching for own profile picture ([e45d9e9](https://github.com/pocket-id/pocket-id/commit/e45d9e970d327a5120ff9fb0c8d42df8af69bb38) by @stonith404)
- emails are considered as medium spam by rspamd ([#337](https://github.com/pocket-id/pocket-id/pull/337) by @alexlehm)

### Other

- add separate worfklow for unit tests ([26e0594](https://github.com/pocket-id/pocket-id/commit/26e05947fe336a08c8b2ef460f768dfe53754e00) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.40.0...v0.40.1

## v0.40.0

### Bug Fixes

- missing write permissions on scripts ([ec4b41a](https://github.com/pocket-id/pocket-id/commit/ec4b41a1d26ea00bb4a95f654ac4cc745b2ce2e8) by @stonith404)

### Features

- allow setting path where keys are stored ([#327](https://github.com/pocket-id/pocket-id/pull/327) by @ItalyPaleAle)

### Other

- add Dev Container ([#313](https://github.com/pocket-id/pocket-id/pull/313) by @nebula-it)
- bump golang.org/x/net from 0.34.0 to 0.36.0 in /backend in the go_modules group across 1 directory ([#326](https://github.com/pocket-id/pocket-id/pull/326) by @dependabot[bot])
- bump @babel/runtime from 7.26.7 to 7.26.10 in /frontend in the npm_and_yarn group across 1 directory ([#328](https://github.com/pocket-id/pocket-id/pull/328) by @dependabot[bot])
- automatically detect release type in release script ([a4bfd08](https://github.com/pocket-id/pocket-id/commit/a4bfd08a0f5a800572cdd9f21d4f938d9dd5ec79) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.39.0...v0.40.0

## v0.39.0

### Bug Fixes

- alternative login method link on mobile ([9ef2ddf](https://github.com/pocket-id/pocket-id/commit/9ef2ddf7963c6959992f3a5d6816840534e926e9) by @stonith404)

### Features

- api key authentication ([#291](https://github.com/pocket-id/pocket-id/pull/291) by @kmendell)

### Other

- adapt api key list to new sort behavior ([d1b9f3a](https://github.com/pocket-id/pocket-id/commit/d1b9f3a44e84430101ac544015b8fa6e21b51ed2) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.38.0...v0.39.0

## v0.38.0

### Bug Fixes

- typo in account settings ([#307](https://github.com/pocket-id/pocket-id/pull/307) by @kotx)
- redirection not correctly if signing in with email code ([e5ec264](https://github.com/pocket-id/pocket-id/commit/e5ec264bfd535752565bcc107099a9df5cb8aba7) by @stonith404)

### Features

- add env variable to disable update check ([31198fe](https://github.com/pocket-id/pocket-id/commit/31198feec2ae77dd6673c42b42002871ddd02d37) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.37.0...v0.38.0

## v0.37.0

### Bug Fixes

- add timeout to update check ([04efc36](https://github.com/pocket-id/pocket-id/commit/04efc3611568a0b0127b542b8cc252d9e783af46) by @stonith404)
- make sorting consistent around tables ([8e344f1](https://github.com/pocket-id/pocket-id/commit/8e344f1151628581b637692a1de0e48e7235a22d) by @stonith404)
- add back setup page ([6a8dd84](https://github.com/pocket-id/pocket-id/commit/6a8dd84ca9396ff3369385af22f7e1f081bec2b2) by @stonith404)

### Documentation

- add Discord contact link to issue template ([2ee0bad](https://github.com/pocket-id/pocket-id/commit/2ee0bad2c0c27322c4ef8560235d71ff4a80f535) by @stonith404)

### Features

- increase default item count per page ([a9713cf](https://github.com/pocket-id/pocket-id/commit/a9713cf6a1e3c879dc773889b7983e51bbe3c45b) by @stonith404)
- add ability to sign in with login code ([#271](https://github.com/pocket-id/pocket-id/pull/271) by @Pyxels)

### Other

- fix type errors ([d0da532](https://github.com/pocket-id/pocket-id/commit/d0da532240ab76ce97b32ed1ebffcc1accf54382) by @stonith404)
- fix user group assignment test ([7885ae0](https://github.com/pocket-id/pocket-id/commit/7885ae011c7390ee9f71ddad2d6742b517c1af67) by @stonith404)
- bump the npm_and_yarn group across 1 directory with 3 updates ([#306](https://github.com/pocket-id/pocket-id/pull/306) by @dependabot[bot])

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.36.0...v0.37.0

## v0.36.0

### Bug Fixes

- default sorting on tables ([#299](https://github.com/pocket-id/pocket-id/pull/299) by @kmendell)

### Features

- enable sd_notify support ([#277](https://github.com/pocket-id/pocket-id/pull/277) by @savely-krasovsky)
- display groups on the account page ([#296](https://github.com/pocket-id/pocket-id/pull/296) by @kmendell)

### Other

- add pr docker build ([#293](https://github.com/pocket-id/pocket-id/pull/293) by @kmendell)
- use `github.repository` variable intead of hardcoding the repository name ([66090f3](https://github.com/pocket-id/pocket-id/commit/66090f36a86be9a59e4909839cbe67ef28ee69ba) by @stonith404)
- remove PR docker build action ([37b24be](https://github.com/pocket-id/pocket-id/commit/37b24bed91a5eb2fe6e6db85198db6794c0611a5) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.35.6...v0.36.0

## v0.35.6

### Bug Fixes

- support `LOGIN` authentication method for SMTP ([#292](https://github.com/pocket-id/pocket-id/pull/292) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.35.5...v0.35.6

## v0.35.5

### Bug Fixes

- profile picture orientation if image is rotated with EXIF ([1026ee4](https://github.com/pocket-id/pocket-id/commit/1026ee4f5b5c7fda78b65c94a5d0f899525defd1) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.35.4...v0.35.5

## v0.35.4

### Bug Fixes

- add `groups` scope and claim to well known endpoint ([4bafee4](https://github.com/pocket-id/pocket-id/commit/4bafee4f58f5a76898cf66d6192916d405eea389) by @stonith404)
- support POST for OIDC userinfo endpoint ([1652cc6](https://github.com/pocket-id/pocket-id/commit/1652cc65f3f966d018d81a1ae22abb5ff1b4c47b) by @stonith404)
- profile picture of other user can't be updated ([#273](https://github.com/pocket-id/pocket-id/pull/273) by @Pyxels)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.35.3...v0.35.4

## v0.35.3

### Bug Fixes

- add option to manually select SMTP TLS method ([#268](https://github.com/pocket-id/pocket-id/pull/268) by @kmendell)
- sync error if LDAP user collides with an existing user ([fde951b](https://github.com/pocket-id/pocket-id/commit/fde951b543281fedf9f602abae26b50881e3d157) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.35.2...v0.35.3

## v0.35.2

### Bug Fixes

- updating profile picture of other user updates own profile picture ([887c5e4](https://github.com/pocket-id/pocket-id/commit/887c5e462a50c8fb579ca6804f1a643d8af78fe8) by @stonith404)
- delete profile picture if user gets deleted ([9a167d4](https://github.com/pocket-id/pocket-id/commit/9a167d4076872e5e3e5d78d2a66ef7203ca5261b) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.35.1...v0.35.2

## v0.35.1

### Bug Fixes

- binary profile picture can't be imported from LDAP ([840a672](https://github.com/pocket-id/pocket-id/commit/840a672fc35ca8476caf86d7efaba9d54bce86aa) by @stonith404)
- add validation that `PUBLIC_APP_URL` can't contain a path ([a6ae7ae](https://github.com/pocket-id/pocket-id/commit/a6ae7ae28713f7fc8018ae2aa7572986df3e1a5b) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.35.0...v0.35.1

## v0.35.0

### Bug Fixes

- app config strings starting with a number are parsed incorrectly ([816c198](https://github.com/pocket-id/pocket-id/commit/816c198a42c189cb1f2d94885d2e3623e47e2848) by @stonith404)
- emails do not get rendered correctly in Gmail ([dca9e7a](https://github.com/pocket-id/pocket-id/commit/dca9e7a11a3ba5d3b43a937f11cb9d16abad2db5) by @stonith404)

### Features

- add ability to upload a profile picture ([#244](https://github.com/pocket-id/pocket-id/pull/244) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.34.0...v0.35.0

## v0.34.0

### Features

- add LDAP group membership attribute ([#236](https://github.com/pocket-id/pocket-id/pull/236) by @kmendell)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.33.0...v0.34.0

## v0.33.0

### Bug Fixes

- show "Sync Now" and "Test Email" button even if UI config is disabled ([4d0fff8](https://github.com/pocket-id/pocket-id/commit/4d0fff821e2245050ce631b4465969510466dfae) by @stonith404)
- alignment of OIDC client details ([c3980d3](https://github.com/pocket-id/pocket-id/commit/c3980d3d28a7158a4dc9369af41f185b891e485e) by @stonith404)
- layout of OIDC client details page on mobile ([3de1301](https://github.com/pocket-id/pocket-id/commit/3de1301fa84b3ab4fff4242d827c7794d44910f2) by @stonith404)

### Features

- add end session endpoint ([#232](https://github.com/pocket-id/pocket-id/pull/232) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.32.0...v0.33.0

## v0.32.0

### Features

- add ability to set custom Geolite DB URL ([#226](https://github.com/pocket-id/pocket-id/pull/226) by @wargio)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.31.0...v0.32.0

## v0.31.0

### Bug Fixes

- user linking in ldap group sync ([#222](https://github.com/pocket-id/pocket-id/pull/222) by @kmendell)

### Features

- display source in user and group table ([#225](https://github.com/pocket-id/pocket-id/pull/225) by @kmendell)
- add ability to override the UI configuration with environment variables ([4e85842](https://github.com/pocket-id/pocket-id/commit/4e858420e9d9713e19f3b35c45c882403717f72f) by @stonith404)
- add warning for only having one passkey configured ([#220](https://github.com/pocket-id/pocket-id/pull/220) by @kmendell)

### Other

- remove Docker Hub registry ([7fbc356](https://github.com/pocket-id/pocket-id/commit/7fbc356d8d1175b087d881764704206540d1ba1d) by @stonith404)
- downgrade ubuntu version of Docker build action runner ([43790dc](https://github.com/pocket-id/pocket-id/commit/43790dc1be76eb1e21e39a2de77da2232ed458b5) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.30.0...v0.31.0

## v0.30.0

### Documentation

- fix freshrss callback url ([#212](https://github.com/pocket-id/pocket-id/pull/212) by @RobinMicek)
- add landing page ([#203](https://github.com/pocket-id/pocket-id/pull/203) by @kmendell)
- improve landing page ([3dda2e1](https://github.com/pocket-id/pocket-id/commit/3dda2e16e98e4f49e48e5ed56d9f471701abd842) by @stonith404)
- add docs root path redirection ([98add37](https://github.com/pocket-id/pocket-id/commit/98add37390000c32d2d74e8b7ee3b2bf9ae15f06) by @stonith404)
- improve mobile layout of landing page ([7c04bda](https://github.com/pocket-id/pocket-id/commit/7c04bda5b77ae753b4bb5bc6a3b336ab8983435d) by @stonith404)

### Features

- add custom ldap search filters ([#216](https://github.com/pocket-id/pocket-id/pull/216) by @kmendell)
- update host configuration to allow external access ([#218](https://github.com/pocket-id/pocket-id/pull/218) by @jonasclaes)

### Other

- fix old docker image references ([0bae7e4](https://github.com/pocket-id/pocket-id/commit/0bae7e4f53d85f8f3f30f80675b8346cc0125a18) by @stonith404)
- add missing permissions to "Build and Push Docker Image" ([d66cf70](https://github.com/pocket-id/pocket-id/commit/d66cf70d50f68811afbb86742b0c108b5d8567fa) by @stonith404)
- remove docs from repository ([0751540](https://github.com/pocket-id/pocket-id/commit/0751540d7d47851698be5d0fde1c330ea24a6d54) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.29.0...v0.30.0

## v0.29.0

### Documentation

- enhance documentation ([#205](https://github.com/pocket-id/pocket-id/pull/205) by @kmendell)

### Features

- add option to disable Caddy in the Docker container ([e864d5d](https://github.com/pocket-id/pocket-id/commit/e864d5dcbff1ef28dc6bf120e4503093a308c5c8) by @stonith404)
- add JSON support in custom claims ([15cde6a](https://github.com/pocket-id/pocket-id/commit/15cde6ac66bc857ac28df545a37c1f4341977595) by @stonith404)

### Other

- replace `stonith404` with `pocket-id` after org migration ([c6ab2b2](https://github.com/pocket-id/pocket-id/commit/c6ab2b252cfca1939f891008b4b93a826c6fc14b) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.28.1...v0.29.0

## v0.28.1

### Bug Fixes

- don't return error page if version info fetching failed ([d06257e](https://github.com/pocket-id/pocket-id/commit/d06257ec9b5e46e25e40c174b4bef02dca0a1ea3) by @stonith404)

### Documentation

- fix reauthentication in caddy-security example ([19ef483](https://github.com/pocket-id/pocket-id/commit/19ef4833e927b9bf4984b43913a39ed58a45a98f) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.28.0...v0.28.1

## v0.28.0

### Bug Fixes

- use cursor pointer on clickable elements ([7798580](https://github.com/pocket-id/pocket-id/commit/77985800ae9628104e03e7f2e803b7ed9eaaf4e0) by @stonith404)
- trusted_proxies for IPv6 enabled hosts ([#189](https://github.com/pocket-id/pocket-id/pull/189) by @apearson)
- non LDAP user group can't be updated after update ([ecd74b7](https://github.com/pocket-id/pocket-id/commit/ecd74b794f1ffb7da05bce0046fb8d096b039409) by @stonith404)
- missing user service dependency ([61e71ad](https://github.com/pocket-id/pocket-id/commit/61e71ad43b8f0f498133d3eb2381382e7bc642b9) by @stonith404)

### Documentation

- add version label to navbar ([#186](https://github.com/pocket-id/pocket-id/pull/186) by @kmendell)
- Add Immich and Headscale client examples ([#191](https://github.com/pocket-id/pocket-id/pull/191) by @jeffreygarc)
- Added Gitea and Memos example ([#194](https://github.com/pocket-id/pocket-id/pull/194) by @PrtmPhlp)
- add custom `pocket-id.org` domain ([e607fe4](https://github.com/pocket-id/pocket-id/commit/e607fe424ae775f93b9bdcee82fbcc421578de67) by @stonith404)
- add new `demo.pocket-id.org` domain to the README ([2d3cba6](https://github.com/pocket-id/pocket-id/commit/2d3cba63089ed31276a29342a39c2a986f158a5a) by @stonith404)
- add helper scripts install for proxmox ([#197](https://github.com/pocket-id/pocket-id/pull/197) by @kmendell)
- add example for adding Pocket ID to FreshRSS ([#200](https://github.com/pocket-id/pocket-id/pull/200) by @UncleArya)

### Features

- allow LDAP users and groups to be deleted if LDAP gets disabled ([9ab1787](https://github.com/pocket-id/pocket-id/commit/9ab178712aa3cc71546a89226e67b7ba91245251) by @stonith404)
- map allowed groups to OIDC clients ([#202](https://github.com/pocket-id/pocket-id/pull/202) by @stonith404)

### Other

- add auto deployment for docs website ([7497f4a](https://github.com/pocket-id/pocket-id/commit/7497f4ad409aeacb5c8f0764a9a91c7e26b4f6d0) by @stonith404)
- ignore irrelevant paths for e2e tests ([a1b20f0](https://github.com/pocket-id/pocket-id/commit/a1b20f0e743314627c38e20ad1d9751d72f3525b) by @stonith404)
- run formatter ([28346da](https://github.com/pocket-id/pocket-id/commit/28346da7319b8b27c8dbe727e8368dc4024e2908) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.27.2...v0.28.0

## v0.27.2

### Bug Fixes

- smtp hello for tls connections ([#180](https://github.com/pocket-id/pocket-id/pull/180) by @kmendell)

### Documentation

- fix typos and improve clarity in proxmox.md ([#183](https://github.com/pocket-id/pocket-id/pull/183) by @BrutalCoding)
- add missing env file flag to frontend start command ([a65ce56](https://github.com/pocket-id/pocket-id/commit/a65ce56b42a0395538884c2dfe0c9454c9b70b9f) by @stonith404)

### Other

- upgrade to Nodejs 22 ([8cd834a](https://github.com/pocket-id/pocket-id/commit/8cd834a503e4df01e3783cc4955e32263d6d87ed) by @stonith404)
- upgrade to Tailwind 4 ([5c452ce](https://github.com/pocket-id/pocket-id/commit/5c452ceef06e76c7d442c29d110ec613b0bb7972) by @stonith404)
- upgrade frontend and backend dependencies ([04c7f18](https://github.com/pocket-id/pocket-id/commit/04c7f180de2f9963d614058975c8ff79e2f0bbab) by @stonith404)
- downgrade formsnap ([dab37c5](https://github.com/pocket-id/pocket-id/commit/dab37c5967999d2e6275eebb7351193d2cc65048) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.27.1...v0.27.2

## v0.27.1

### Bug Fixes

- send hostname derived from `PUBLIC_APP_URL` with SMTP EHLO command ([397544c](https://github.com/pocket-id/pocket-id/commit/397544c0f3f2b49f1f34ae53e6b9daf194d1ae28) by @stonith404)
- use OS hostname for SMTP EHLO message ([47c39f6](https://github.com/pocket-id/pocket-id/commit/47c39f6d382c496cb964262adcf76cc8dbb96da3) by @stonith404)
- add `__HOST` prefix to cookies ([#175](https://github.com/pocket-id/pocket-id/pull/175) by @stonith404)

### Documentation

- add more client-examples ([#166](https://github.com/pocket-id/pocket-id/pull/166) by @kmendell)
- remove duplicate `contribute.md` ([d071641](https://github.com/pocket-id/pocket-id/commit/d0716418908470e7408669153d508cd05a5d4c51) by @stonith404)
- make CONTRIBUTING instructions work & fix example envs ([#152](https://github.com/pocket-id/pocket-id/pull/152) by @cdanis)

### Other

- add GitHub release creation to `create-release.sh` script ([7b40355](https://github.com/pocket-id/pocket-id/commit/7b403552ba91f28b38c8e30ed46e13643ed1b876) by @stonith404)
- bug template update ([#133](https://github.com/pocket-id/pocket-id/pull/133) by @kmendell)
- remove duplicate text from issue template ([2884021](https://github.com/pocket-id/pocket-id/commit/2884021055325df15f2b87a37872dd868b0219ca) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.27.0...v0.27.1

## v0.27.0

### Bug Fixes

- ensure the downloaded GeoLite2 DB is not corrupted & prevent RW race condition ([#138](https://github.com/pocket-id/pocket-id/pull/138) by @wargio)
- add save changes dialog before sending test email ([#165](https://github.com/pocket-id/pocket-id/pull/165) by @kmendell)

### Documentation

- create sample-configurations.md ([#142](https://github.com/pocket-id/pocket-id/pull/142) by @kamilkosek)
- add `delay_start` to caddy security ([c211d3f](https://github.com/pocket-id/pocket-id/commit/c211d3fc67a17fc1ed6e207fecb5c29e1d3542c5) by @stonith404)
- add docusaurus docs ([#118](https://github.com/pocket-id/pocket-id/pull/118) by @kmendell)
- fix open-webui docs page ([#162](https://github.com/pocket-id/pocket-id/pull/162) by @kmendell)

### Features

- display private IP ranges correctly in audit log ([#139](https://github.com/pocket-id/pocket-id/pull/139) by @cdanis)

### Other

- optimize images ([#161](https://github.com/pocket-id/pocket-id/pull/161) by @imgbot[bot])

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.26.0...v0.27.0

## v0.26.0

### Bug Fixes

- non LDAP users get created with a empty LDAP ID string ([3f02d08](https://github.com/pocket-id/pocket-id/commit/3f02d081098ad2caaa60a56eea4705639f80d01f) by @stonith404)

### Features

- support wildcard callback URLs ([8a1db0c](https://github.com/pocket-id/pocket-id/commit/8a1db0cb4a5d4b32b4fdc19d41fff688a7c71a56) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.25.1...v0.26.0

## v0.25.1

### Bug Fixes

- disable account details inputs if user is imported from LDAP ([a8b9d60](https://github.com/pocket-id/pocket-id/commit/a8b9d60a86e80c10d6fba07072b1d32cec400ecb) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.25.0...v0.25.1

## v0.25.0

### Bug Fixes

- search input not displayed if response hasn't any items ([05a98eb](https://github.com/pocket-id/pocket-id/commit/05a98ebe87d7a88e8b96b144c53250a40d724ec3) by @stonith404)
- always set secure on cookie ([#130](https://github.com/pocket-id/pocket-id/pull/130) by @cdanis)
- session duration ignored in cookie expiration ([bc8f454](https://github.com/pocket-id/pocket-id/commit/bc8f454ea173ecc60e06450a1d22e24207f76714) by @stonith404)
- don't panic if LDAP sync fails on startup ([e284e35](https://github.com/pocket-id/pocket-id/commit/e284e352e2b95fac1d098de3d404e8531de4b869) by @stonith404)
- improve spacing of checkboxes on application configuration page ([090eca2](https://github.com/pocket-id/pocket-id/commit/090eca202d198852e6fbf4e6bebaf3b5ada13944) by @stonith404)

### Documentation

- add guide to setup Pocket ID with Caddy ([6e3728d](https://github.com/pocket-id/pocket-id/commit/6e3728ddc86cce95b2041513564b087463f5b2d3) by @stonith404)

### Features

- add LDAP sync ([#106](https://github.com/pocket-id/pocket-id/pull/106) by @kmendell)
- allow sign in with email ([#100](https://github.com/pocket-id/pocket-id/pull/100) by @stonith404)
- automatically authorize client if signed in ([d5dd118](https://github.com/pocket-id/pocket-id/commit/d5dd118a3f4ad6eed9ca496c458201bb10f148a0) by @stonith404)

### Other

- run formatter ([692ff70](https://github.com/pocket-id/pocket-id/commit/692ff70c918de47463d0a98ba365883f24630968) by @stonith404)
- adapt OIDC tests ([d4055af](https://github.com/pocket-id/pocket-id/commit/d4055af3f4dbd3c3cfa0475f2afa1fbb24dd565a) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.24.1...v0.25.0

## v0.24.1

### Bug Fixes

- audit log table overflow if row data is long ([4d337a2](https://github.com/pocket-id/pocket-id/commit/4d337a20c5cb92ef80bb7402f9b99b08e3ad0b6b) by @stonith404)
- optional arguments not working with `create-one-time-access-token.sh` ([8885571](https://github.com/pocket-id/pocket-id/commit/888557171d61589211b10f70dce405126216ad61) by @stonith404)
- remove restrictive validation for group names ([be6e25a](https://github.com/pocket-id/pocket-id/commit/be6e25a167de8bf07075b46f09d9fc1fa6c74426) by @stonith404)

### Documentation

- add account recovery to README ([2a984ee](https://github.com/pocket-id/pocket-id/commit/2a984eeaf1ee169f9f4987acde0e79660d2b6781) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.24.0...v0.24.1

## v0.24.0

### Bug Fixes

- send test email to the user that has requested it ([a649c4b](https://github.com/pocket-id/pocket-id/commit/a649c4b4a543286123f4d1f3c411fe1a7e2c6d71) by @stonith404)
- pkce state not correctly reflected in oidc client info ([61d18a9](https://github.com/pocket-id/pocket-id/commit/61d18a9d1b167ff59a59523ff00d00ca8f23258d) by @stonith404)

### Features

- add sorting for tables ([fd69830](https://github.com/pocket-id/pocket-id/commit/fd69830c2681985e4fd3c5336a2b75c9fb7bc5d4) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.23.0...v0.24.0

## v0.23.0

### Features

- add PKCE for non public clients ([adcf3dd](https://github.com/pocket-id/pocket-id/commit/adcf3ddc6682794e136a454ef9e69ddd130626a8) by @stonith404)
- use same table component for OIDC client list as all other lists ([2d31fc2](https://github.com/pocket-id/pocket-id/commit/2d31fc2cc9201bb93d296faae622f52c6dcdfebc) by @stonith404)

### Other

- include static assets in binary ([785200d](https://github.com/pocket-id/pocket-id/commit/785200de61deb1544aac5ff6f914a35e27632bbc) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.22.0...v0.23.0

## v0.22.0

### Bug Fixes

- passkey can't be added if `PUBLIC_APP_URL` includes a port ([0729ce9](https://github.com/pocket-id/pocket-id/commit/0729ce9e1a8dab9912900a01dcd0fbd892718a1a) by @stonith404)
- hash in callback url is incorrectly appended ([f6f2736](https://github.com/pocket-id/pocket-id/commit/f6f2736bba65eee017f2d8cdaa70621574092869) by @stonith404)
- allow first and last name of user to be between 1 and 50 characters ([1ff20ca](https://github.com/pocket-id/pocket-id/commit/1ff20caa3ccd651f9fb30f958ffb807dfbbcbd8a) by @stonith404)
- make user validation consistent between pages ([333a1a1](https://github.com/pocket-id/pocket-id/commit/333a1a18d59f675111f4ed106fa5614ef563c6f4) by @stonith404)

### Documentation

- improve text in README ([ff75322](https://github.com/pocket-id/pocket-id/commit/ff75322e7de08970d8573cd3f081194305b0daee) by @stonith404)
- add "groups" scope to the oauth2-proxy sample configuration ([#85](https://github.com/pocket-id/pocket-id/pull/85) by @janpfischer)

### Features

- add warning if passkeys missing ([2d0bd8d](https://github.com/pocket-id/pocket-id/commit/2d0bd8dcbfb73650b7829cb66f40decb284bd73b) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.21.0...v0.22.0

## v0.21.0

### Bug Fixes

- OIDC client logo gets removed if other properties get updated ([789d939](https://github.com/pocket-id/pocket-id/commit/789d9394a533831e7e2fb8dc3f6b338787336ad8) by @stonith404)

### Features

- improve error state design for login page ([0716c38](https://github.com/pocket-id/pocket-id/commit/0716c38fb8ce7fa719c7fe0df750bdb213786c21) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.20.1...v0.21.0

## v0.20.1

### Bug Fixes

- `create-one-time-access-token.sh` script not compatible with postgres ([34e3519](https://github.com/pocket-id/pocket-id/commit/34e35193f9f3813f6248e60f15080d753e8da7ae) by @stonith404)
- wrong date time datatype used for read operations with Postgres ([bad901e](https://github.com/pocket-id/pocket-id/commit/bad901ea2b661aadd286e5e4bed317e73bd8a70d) by @stonith404)

### Other

- add e2e test for one time access tokens ([5480ab0](https://github.com/pocket-id/pocket-id/commit/5480ab0f188ed76a991b05ebc81242a688a39a5f) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.20.0...v0.20.1

## v0.20.0

### Documentation

- add callback url to `proxy-services.md` ([3006bc9](https://github.com/pocket-id/pocket-id/commit/3006bc9ef798189c1f1311ae1e832055d8653e51) by @stonith404)
- add ghcr.io Docker image to `docker-compose.yml` ([e9d83dd](https://github.com/pocket-id/pocket-id/commit/e9d83dd6c3d2c9d053271cf24b23cc9228892bd4) by @stonith404)

### Features

- add support for Postgres database provider ([#79](https://github.com/pocket-id/pocket-id/pull/79) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.19.0...v0.20.0

## v0.19.0

### Documentation

- add demo link ([9a8ec15](https://github.com/pocket-id/pocket-id/commit/9a8ec1567851159bed938a2aad9b79c299816b4b) by @stonith404)

### Features

- add Tailscale IP detection with CGNAT range check ([#77](https://github.com/pocket-id/pocket-id/pull/77) by @s0up4200)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.18.0...v0.19.0

## v0.18.0

### Bug Fixes

- email save toast shows two times ([f2bfc73](https://github.com/pocket-id/pocket-id/commit/f2bfc731585ad7424eb8c4c41c18368fc0f75ffc) by @stonith404)

### Documentation

- improve `MAXMIND_LICENSE_KEY` documentation in readme ([31a6b57](https://github.com/pocket-id/pocket-id/commit/31a6b57ec1b795304d72aab0c5693d340620c533) by @stonith404)
- add `PUID` and `PGID` to `.env.example` ([7d6b1d1](https://github.com/pocket-id/pocket-id/commit/7d6b1d19e99ed9626fe4067337f20d65acda2527) by @stonith404)

### Features

- add option to disable TLS for email sending ([f9fa2c6](https://github.com/pocket-id/pocket-id/commit/f9fa2c6706a8bf949fe5efd6664dec8c80e18659) by @stonith404)
- allow empty user and password in SMTP configuration ([a9f4dad](https://github.com/pocket-id/pocket-id/commit/a9f4dada321841d3611b15775307228b34e7793f) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.17.0...v0.18.0

## v0.17.0

### Bug Fixes

- don't try to create a new user if the Docker user is not root ([#71](https://github.com/pocket-id/pocket-id/pull/71) by @cdanis)

### Documentation

- fix OAuth2 proxy link in readme ([0b4101c](https://github.com/pocket-id/pocket-id/commit/0b4101ccce973847d863e441816f77912def388a) by @stonith404)

### Features

- add option to specify the Max Mind license key for the Geolite2 db ([fcf08a4](https://github.com/pocket-id/pocket-id/commit/fcf08a4d898160426442bd80830f4431988f4313) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.16.0...v0.17.0

## v0.16.0

### Features

- add health check ([058084e](https://github.com/pocket-id/pocket-id/commit/058084ed64816b12108e25bf04af988fc97772ed) by @stonith404)
- improve error message for invalid callback url ([#69](https://github.com/pocket-id/pocket-id/pull/69) by @alexlehm)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.15.0...v0.16.0

## v0.15.0

### Bug Fixes

- mobile layout overflow on application configuration page ([e784093](https://github.com/pocket-id/pocket-id/commit/e784093342f9977ea08cac65ff0c3de4d2644872) by @stonith404)

### Documentation

- add info that PKCE isn't implemented yet ([760c8e8](https://github.com/pocket-id/pocket-id/commit/760c8e83bb5a2362e1bdc21f2f0b92c154783c50) by @stonith404)

### Features

- add PKCE support ([3613ac2](https://github.com/pocket-id/pocket-id/commit/3613ac261cf65a2db0620ff16dc6df239f6e5ecd) by @stonith404)
- add option to skip TLS certificate check and ability to send test email ([653d948](https://github.com/pocket-id/pocket-id/commit/653d948f73b61e6d1fd3484398fef1a2a37e6d92) by @stonith404)

### Other

- add Docker image to ghcr.io and add Docker metadata action ([5f44fef](https://github.com/pocket-id/pocket-id/commit/5f44fef85f9227f73ffde8e645d2a1631b00174d) by @stonith404)
- move checkboxes with label in seperate component ([a1302ef](https://github.com/pocket-id/pocket-id/commit/a1302ef7bf5b06b68c08e5bb8de0f4472c660774) by @stonith404)
- make Docker image run without root user ([#67](https://github.com/pocket-id/pocket-id/pull/67) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.14.0...v0.15.0

## v0.14.0

### Bug Fixes

- time displayed incorrectly in audit log ([3d3fb4d](https://github.com/pocket-id/pocket-id/commit/3d3fb4d855ef510f2292e98fcaaaf83debb5d3e0) by @stonith404)
- overflow of pagination control on mobile ([de45398](https://github.com/pocket-id/pocket-id/commit/de4539890349153c467013c24c4d6b30feb8fed8) by @stonith404)

### Features

- add audit log event for one time access token sign in ([aca2240](https://github.com/pocket-id/pocket-id/commit/aca2240a50a12e849cfb6e1aa56390b000aebae0) by @stonith404)

### Other

- fix build warnings ([725388f](https://github.com/pocket-id/pocket-id/commit/725388fcc7b98b4461354d2676a39ff34e50d6e1) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.13.1...v0.14.0

## v0.13.1

### Bug Fixes

- typo in Self-Account Editing description ([5b9f4d7](https://github.com/pocket-id/pocket-id/commit/5b9f4d732615f428c13d3317da96a86c5daebd89) by @stonith404)
- errors in middleware do not abort the request ([376d747](https://github.com/pocket-id/pocket-id/commit/376d747616b1e835f252d20832c5ae42b8b0b737) by @stonith404)

### Features

- add list empty indicator ([becfc00](https://github.com/pocket-id/pocket-id/commit/becfc0004a87c01e18eb92ac85bf4e33f105b6a3) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.13.0...v0.13.1

## v0.13.0

### Bug Fixes

- bad gateway error if nginx reverse proxy is in front ([590cb02](https://github.com/pocket-id/pocket-id/commit/590cb02f6c15f3e8aae81d67edc8770a2cffdb42) by @stonith404)

### Documentation

- add Jellyfin Integration Guide ([#51](https://github.com/pocket-id/pocket-id/pull/51) by @donkevlar)
- add nginx configuration to README ([78c88f5](https://github.com/pocket-id/pocket-id/commit/78c88f53396d056b6a8cb388c0de9649db072066) by @stonith404)

### Features

- add ability to define expiration of one time link ([2ccabf8](https://github.com/pocket-id/pocket-id/commit/2ccabf835c2c923d6986d9cafb4e878f5110b91a) by @stonith404)

### Other

- change default port in dockerfile ([3484daf](https://github.com/pocket-id/pocket-id/commit/3484daf8706a6122de6a04af2595932b224afa99) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.12.0...v0.13.0

## v0.12.0

### Features

- custom claims ([#53](https://github.com/pocket-id/pocket-id/pull/53) by @stonith404)
- add validation to custom claim input ([7bfc3f4](https://github.com/pocket-id/pocket-id/commit/7bfc3f43a591287c038187ed5e782de6b9dd738b) by @stonith404)
- add option to disable self-account editing ([8304065](https://github.com/pocket-id/pocket-id/commit/83040656525cf7b6c8f2acf416c5f8f3288f3d48) by @stonith404)

### Other

- fix flaky playwright tests ([735dc70](https://github.com/pocket-id/pocket-id/commit/735dc70d5fd16abe6e9a109dbc8e190ebbb819b1) by @stonith404)
- fix html reporting of playwright ([0b0a678](https://github.com/pocket-id/pocket-id/commit/0b0a6781ff4bfa226ef77bfda19fc9cb878720ae) by @stonith404)
- correctly reset app config in tests ([3350398](https://github.com/pocket-id/pocket-id/commit/3350398abcf948243a8445225791127a99e4095e) by @stonith404)
- fix custom claims test data ([b9daa5d](https://github.com/pocket-id/pocket-id/commit/b9daa5d7576881c5b5fbc1f290661835bfd8a892) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.11.0...v0.12.0

## v0.11.0

### Bug Fixes

- powered by link text color in light mode ([18c5103](https://github.com/pocket-id/pocket-id/commit/18c5103c20ce79abdc0f724cdedd642c09269e78) by @stonith404)

### Features

- add `email_verified` claim ([5565f60](https://github.com/pocket-id/pocket-id/commit/5565f60d6d62ca24bedea337e21effc13e5853a5) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.10.0...v0.11.0

## v0.10.0

### Bug Fixes

- increase callback url count ([f3f0e1d](https://github.com/pocket-id/pocket-id/commit/f3f0e1d56d7656bdabbd745a4eaf967f63193b6c) by @stonith404)
- improve text for initial admin account setup ([0a07344](https://github.com/pocket-id/pocket-id/commit/0a0734413943b1fff27d8f4ccf07587e207e2189) by @stonith404)
- no DTO was returned from exchange one time access token endpoint ([824c5cb](https://github.com/pocket-id/pocket-id/commit/824c5cb4f3d6be7f940c1758112fbe9322df5768) by @stonith404)
- cache version information for 3 hours ([29d632c](https://github.com/pocket-id/pocket-id/commit/29d632c1514d6edacdfebe6deae4c95fc5a0f621) by @stonith404)

### Features

- add version information to footer and update link if new update is available ([70ad0b4](https://github.com/pocket-id/pocket-id/commit/70ad0b4f39699fd81ffdfd5c8d6839f49348be78) by @stonith404)
- add script for creating one time access token ([a1985ce](https://github.com/pocket-id/pocket-id/commit/a1985ce1b200550e91c5cb42a8d19899dcec831e) by @stonith404)

### Other

- save dates as unix timestamps in database ([b39bc4f](https://github.com/pocket-id/pocket-id/commit/b39bc4f79a87c7d2a47e57705a99bb8fadcdde5d) by @stonith404)
- move development scripts into seperate folder ([3a300a2](https://github.com/pocket-id/pocket-id/commit/3a300a2b51be9516d8ff415e0a79f9254a2485e1) by @stonith404)
- improve check of required tools in one time access token script ([0aff618](https://github.com/pocket-id/pocket-id/commit/0aff6181c9ae7b5dcfa8b5f66afe61390362b533) by @stonith404)
- dump frontend dependencies ([2092007](https://github.com/pocket-id/pocket-id/commit/2092007752d3442d1af4dc79190cb50a1ad97cb5) by @stonith404)
- fix wrong file name of package.json in release script ([6560fd9](https://github.com/pocket-id/pocket-id/commit/6560fd92795e3f1006707a4c77ae70d10e1ff258) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.9.0...v0.10.0

## v0.9.0

### Bug Fixes

- allow copy to clipboard for client secret ([29748cc](https://github.com/pocket-id/pocket-id/commit/29748cc6c7b7e5a6b54bfe837e0b1a98fa1ad594) by @stonith404)

### Features

- use improve table for users and audit logs ([11ed661](https://github.com/pocket-id/pocket-id/commit/11ed661f86a512f78f66d604a10c1d47d39f2c39) by @stonith404)
- add environment variable to change the caddy port in Docker ([ff06bf0](https://github.com/pocket-id/pocket-id/commit/ff06bf0b34496ce472ba6d3ebd4ea249f21c0ec3) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.8.1...v0.9.0

## v0.8.1

### Bug Fixes

- add key id to JWK ([282ff82](https://github.com/pocket-id/pocket-id/commit/282ff82b0c7e2414b3528c8ca325758245b8ae61) by @stonith404)

### Other

- create dummy GeoLite2 City database for e2e tests ([896da81](https://github.com/pocket-id/pocket-id/commit/896da812a3fe0cad89305f793c405eec0d6b5cfa) by @stonith404)
- dump dependencies ([9d5f83d](https://github.com/pocket-id/pocket-id/commit/9d5f83da78d42540260775a4e626c006b5f331c8) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.8.0...v0.8.1

## v0.8.0

### Features

- add location based on ip to the audit log ([025378d](https://github.com/pocket-id/pocket-id/commit/025378d14edd2d72da76e90799a0ccdd42cf672c) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.7.1...v0.8.0

## v0.7.1

### Bug Fixes

- initials don't get displayed if Gravatar avatar doesn't exist ([e095628](https://github.com/pocket-id/pocket-id/commit/e09562824a794bc7d240e9d229709d4b389db7d5) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.7.0...v0.7.1

## v0.7.0

### Features

- add ability to set light and dark mode logo ([be45eed](https://github.com/pocket-id/pocket-id/commit/be45eed125e33e9930572660a034d5f12dc310ce) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.6.0...v0.7.0

## v0.6.0

### Bug Fixes

- only return user groups if it is explicitly requested ([a4a90a1](https://github.com/pocket-id/pocket-id/commit/a4a90a16a9726569a22e42560184319b25fd7ca6) by @stonith404)

### Features

- add user groups ([24c948e](https://github.com/pocket-id/pocket-id/commit/24c948e6a66f283866f6c8369c16fa6cbcfa626c) by @stonith404)
- add gravatar profile picture integration ([365734e](https://github.com/pocket-id/pocket-id/commit/365734ec5d8966c2ab877c60cfb176b9cdc36880) by @stonith404)
- add copy to clipboard option for OIDC client information ([f82020c](https://github.com/pocket-id/pocket-id/commit/f82020ccfb0d4fbaa1dd98182188149d8085252a) by @stonith404)

### Other

- format caddyfiles ([7a54d3a](https://github.com/pocket-id/pocket-id/commit/7a54d3ae2085beae0c5b565e2873e78457307901) by @stonith404)
- add user group tests ([d02d893](https://github.com/pocket-id/pocket-id/commit/d02d8931a0c2510c3e8ec354ae634aac970cba2b) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.5.3...v0.6.0

## v0.5.3

### Bug Fixes

- port environment variables get ignored in caddyfile ([3c67765](https://github.com/pocket-id/pocket-id/commit/3c67765992d7369a79812bc8cd216c9ba12fd96e) by @stonith404)
- add space to "Firstname" and "Lastname" label ([#31](https://github.com/pocket-id/pocket-id/pull/31) by @edbourque0)

### Other

- set the go version to `1.23.1` ([6bb613e](https://github.com/pocket-id/pocket-id/commit/6bb613e0e7979e253dd0bfade8c24d7409add617) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.5.2...v0.5.3

## v0.5.2

### Bug Fixes

- updated application name doesn't apply to webauthn credential ([924bb14](https://github.com/pocket-id/pocket-id/commit/924bb1468bbd8e42fa6a530ef740be73ce3b3914) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.5.1...v0.5.2

## v0.5.1

### Bug Fixes

- debounce oidc client and user search ([9c2848d](https://github.com/pocket-id/pocket-id/commit/9c2848db1d93c230afc6c5f64e498e9f6df8c8a7) by @stonith404)

### Features

- improve email templating ([#27](https://github.com/pocket-id/pocket-id/pull/27) by @oidq)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.5.0...v0.5.1

## v0.5.0

### Features

- add audit log with email notification ([#26](https://github.com/pocket-id/pocket-id/pull/26) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.4.1...v0.5.0

## v0.4.1

### Bug Fixes

- limit width of content on large screens ([c6f83a5](https://github.com/pocket-id/pocket-id/commit/c6f83a581ad385391d77fec7eeb385060742f097) by @stonith404)
- show error message if error occurs while authorizing new client ([8038a11](https://github.com/pocket-id/pocket-id/commit/8038a111dd7fa8f5d421b29c3bc0c11d865dc71b) by @stonith404)

### Features

- add name claim to userinfo endpoint and id token ([4e7574a](https://github.com/pocket-id/pocket-id/commit/4e7574a297307395603267c7a3285d538d4111d8) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.4.0...v0.4.1

## v0.4.0

### Bug Fixes

- oidc client logo not displayed on authorize page ([28ed064](https://github.com/pocket-id/pocket-id/commit/28ed064668afeec8f80adda59ba94f1fc2fbce17) by @stonith404)
- typo in hasLogo property of oidc dto ([2b9413c](https://github.com/pocket-id/pocket-id/commit/2b9413c7575e1322f8547490a9b02a1836bad549) by @stonith404)
- non pointer passed to create user ([e7861df](https://github.com/pocket-id/pocket-id/commit/e7861df95a6beecab359d1c56f4383373f74bb73) by @stonith404)

### Features

- add setup details to oidc client details ([fd21ce5](https://github.com/pocket-id/pocket-id/commit/fd21ce5aac1daeba04e4e7399a0720338ea710c2) by @stonith404)
- add support for more username formats ([903b0b3](https://github.com/pocket-id/pocket-id/commit/903b0b39181c208e9411ee61849d2671e7c56dc5) by @stonith404)

### Other

- rename user service ([8e27320](https://github.com/pocket-id/pocket-id/commit/8e27320649334e632b6dc1bbd89125b9d2f01531) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.3.1...v0.4.0

## v0.3.1

### Bug Fixes

- empty lists don't get returned correctly from the api ([97f7fc4](https://github.com/pocket-id/pocket-id/commit/97f7fc4e288c2bb49210072a7a151b58ef44f5b5) by @stonith404)

### Other

- upgrade dependencies ([fc47c2a](https://github.com/pocket-id/pocket-id/commit/fc47c2a2a4b01a9e97c6c81de06ffbba99f6e639) by @stonith404)
- fix missing host in cleanup request ([6769cc8](https://github.com/pocket-id/pocket-id/commit/6769cc8c10bba3c7a06cc00b3b10a21424aa98c7) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.3.0...v0.3.1

## v0.3.0

### Bug Fixes

- db migration for multiple callback urls ([552d7cc](https://github.com/pocket-id/pocket-id/commit/552d7ccfa58d7922ecb94bdfe6a86651b4cf2745) by @stonith404)

### Documentation

- add proxy guide ([9f49e55](https://github.com/pocket-id/pocket-id/commit/9f49e5577effb2417e2ab06d663e70c91d4dce35) by @stonith404)
- compress screenshot in README ([16f273f](https://github.com/pocket-id/pocket-id/commit/16f273ffceeed5426f64146282ce76fbec150ad7) by @stonith404)

### Features

- add support for multiple callback urls ([8166e2e](https://github.com/pocket-id/pocket-id/commit/8166e2ead7fc71a0b7a45950b05c5c65a60833b6) by @stonith404)

### Other

- use dtos in controllers ([ae7aeb0](https://github.com/pocket-id/pocket-id/commit/ae7aeb0945c00aa9082d68790080faa077b63749) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.2.1...v0.3.0

## v0.2.1

### Bug Fixes

- session duration can't be updated ([4780548](https://github.com/pocket-id/pocket-id/commit/478054884389ed8a08d707fd82da7b31177a67e5) by @stonith404)

### Other

- fix update general configuration test ([aaed71e](https://github.com/pocket-id/pocket-id/commit/aaed71e1c8d6884e0f13beaf4c6c29f3460efa13) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.2.0...v0.2.1

## v0.2.0

### Documentation

- add Unraid to README ([b49063d](https://github.com/pocket-id/pocket-id/commit/b49063d692a36fc99d8565db1287385be6ebd2b5) by @stonith404)
- add note that https is required ([74f4c22](https://github.com/pocket-id/pocket-id/commit/74f4c22800a4ef83df9b75de557cf2e624538901) by @stonith404)

### Features

- add `INTERNAL_BACKEND_URL` env variable ([0595d73](https://github.com/pocket-id/pocket-id/commit/0595d73ea5afbd7937b8f292ffe624139f818f41) by @stonith404)
- add user info endpoint to support more oidc clients ([fdc1921](https://github.com/pocket-id/pocket-id/commit/fdc1921f5dcb5ac6beef8d1c9b1b7c53f514cce5) by @stonith404)
- change default logo ([9eec7a3](https://github.com/pocket-id/pocket-id/commit/9eec7a3e9eb7f690099f38a5d4cf7c2516ea9ef9) by @stonith404)

### Other

- use dependency injection in backend ([601f6c4](https://github.com/pocket-id/pocket-id/commit/601f6c488a7b3c266a1d2174282ab3203841a6e5) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.1.3...v0.2.0

## v0.1.3

### Bug Fixes

- logo not white in dark mode ([5749d05](https://github.com/pocket-id/pocket-id/commit/5749d0532fc38bf2fc66571878b7c71643895c9e) by @stonith404)
- add missing passkey flags to make icloud passkeys work ([cc407e1](https://github.com/pocket-id/pocket-id/commit/cc407e17d409041ed88b959ce13bd581663d55c3) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.1.2...v0.1.3

## v0.1.2

### Bug Fixes

- background image not loading ([7b44189](https://github.com/pocket-id/pocket-id/commit/7b4418958ebfffffd216ef5ba7313cfaad9bc9fa) by @stonith404)
- a non admin user was able to make himself an admin ([df0cd38](https://github.com/pocket-id/pocket-id/commit/df0cd38deeea516c47b26a080eed522f19f7290f) by @stonith404)
- disable search engine indexing ([8395492](https://github.com/pocket-id/pocket-id/commit/83954926f5ee328ebf75a75bb47b380ec0680378) by @stonith404)
- background image on mobile ([4a808c8](https://github.com/pocket-id/pocket-id/commit/4a808c86ac204f9b58cfa02f5ceb064162a87076) by @stonith404)

### Features

- add option to change session duration ([475b932](https://github.com/pocket-id/pocket-id/commit/475b932f9d0ec029ada844072e9d89bebd4e902c) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.1.1...v0.1.2

## v0.1.1

### Bug Fixes

- one time link not displayed correctly ([486771f](https://github.com/pocket-id/pocket-id/commit/486771f433872d08164156d5d6fb0aeb5ae0d125) by @stonith404)

### Features

- add rounded corners to logo ([bec908f](https://github.com/pocket-id/pocket-id/commit/bec908f9078aaa4eec03b730fc36b9fffb1ece74) by @stonith404)

### Other

- change docker image tag in `docker-compose.yml` ([bc86020](https://github.com/pocket-id/pocket-id/commit/bc860204e3e1041a77f0d6db70e38fad7dc6eac6) by @stonith404)
- fix typo in docker image ([4534400](https://github.com/pocket-id/pocket-id/commit/4534400d41e55e4d94b17e27fc36ea47710eb7ad) by @stonith404)

**Full Changelog**: https://github.com/pocket-id/pocket-id/compare/v0.1.0...v0.1.1

## v0.1.0
