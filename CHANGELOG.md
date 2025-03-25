## [](https://github.com/pocket-id/pocket-id/compare/v0.43.1...v) (2025-03-25)


### Features

* add OIDC refresh_token support ([#325](https://github.com/pocket-id/pocket-id/issues/325)) ([b8dcda8](https://github.com/pocket-id/pocket-id/commit/b8dcda80497e554d163a370eff81fe000f8831f4))


### Bug Fixes

* hash the refresh token in the DB (security) ([#379](https://github.com/pocket-id/pocket-id/issues/379)) ([8c96381](https://github.com/pocket-id/pocket-id/commit/8c963818bb90c84dac04018eec93790900d4b0ce))
* skip ldap objects without a valid unique id ([#376](https://github.com/pocket-id/pocket-id/issues/376)) ([cdfe816](https://github.com/pocket-id/pocket-id/commit/cdfe8161d4429bdfe879887fe0b563a67c14f50b))
* stop container if Caddy, the frontend or the backend fails ([e6f5019](https://github.com/pocket-id/pocket-id/commit/e6f50191cf05a5d0ac0e0000cf66423646f1920e))

## [](https://github.com/pocket-id/pocket-id/compare/v0.43.0...v) (2025-03-20)


### Bug Fixes

* wrong base locale causes crash ([3120ebf](https://github.com/pocket-id/pocket-id/commit/3120ebf239b90f0bc0a0af33f30622e034782398))

## [](https://github.com/pocket-id/pocket-id/compare/v0.42.1...v) (2025-03-20)


### Features

* add support for translations ([#349](https://github.com/pocket-id/pocket-id/issues/349)) ([269b5a3](https://github.com/pocket-id/pocket-id/commit/269b5a3c9249bb8081c74741141d3d5a69ea42a2))
* **passkeys:** name new passkeys based on agguids ([#332](https://github.com/pocket-id/pocket-id/issues/332)) ([041c565](https://github.com/pocket-id/pocket-id/commit/041c565dc10f15edb3e8ab58e9a4df5e48a2a6d3))

## [](https://github.com/pocket-id/pocket-id/compare/v0.42.0...v) (2025-03-18)


### Bug Fixes

* kid not added to JWTs ([f7e36a4](https://github.com/pocket-id/pocket-id/commit/f7e36a422ea6b5327360c9a13308ae408ff7fffe))

## [](https://github.com/pocket-id/pocket-id/compare/v0.41.0...v) (2025-03-18)


### Features

* store keys as JWK on disk ([#339](https://github.com/pocket-id/pocket-id/issues/339)) ([a7c9741](https://github.com/pocket-id/pocket-id/commit/a7c9741802667811c530ef4e6313b71615ec6a9b))

## [](https://github.com/pocket-id/pocket-id/compare/v0.40.1...v) (2025-03-18)


### Features

* **profile-picture:** allow reset of profile picture ([#355](https://github.com/pocket-id/pocket-id/issues/355)) ([8f14618](https://github.com/pocket-id/pocket-id/commit/8f146188d57b5c08a4c6204674c15379232280d8))


### Bug Fixes

* own avatar not loading ([#351](https://github.com/pocket-id/pocket-id/issues/351)) ([0423d35](https://github.com/pocket-id/pocket-id/commit/0423d354f533d2ff4fd431859af3eea7d4d7044f))

## [](https://github.com/pocket-id/pocket-id/compare/v0.40.0...v) (2025-03-16)


### Bug Fixes

* API keys not working if sqlite is used ([8ead0be](https://github.com/pocket-id/pocket-id/commit/8ead0be8cd0cfb542fe488b7251cfd5274975ae1))
* caching for own profile picture ([e45d9e9](https://github.com/pocket-id/pocket-id/commit/e45d9e970d327a5120ff9fb0c8d42df8af69bb38))
* email logo icon displaying too big ([#336](https://github.com/pocket-id/pocket-id/issues/336)) ([b483e2e](https://github.com/pocket-id/pocket-id/commit/b483e2e92fdb528e7de026350a727d6970227426))
* emails are considered as medium spam by rspamd ([#337](https://github.com/pocket-id/pocket-id/issues/337)) ([39b7f66](https://github.com/pocket-id/pocket-id/commit/39b7f6678c98cadcdc3abfbcb447d8eb0daa9eb0))
* Fixes and performance improvements in utils package ([#331](https://github.com/pocket-id/pocket-id/issues/331)) ([348192b](https://github.com/pocket-id/pocket-id/commit/348192b9d7e2698add97810f8fba53d13d0df018))
* remove custom claim key restrictions ([9f28503](https://github.com/pocket-id/pocket-id/commit/9f28503d6c73d3521d1309bee055704a0507e9b5))

## [](https://github.com/pocket-id/pocket-id/compare/v0.39.0...v) (2025-03-13)


### Features

* allow setting path where keys are stored ([#327](https://github.com/pocket-id/pocket-id/issues/327)) ([7b654c6](https://github.com/pocket-id/pocket-id/commit/7b654c6bd111ddcddd5e3450cbf326d9cf1777b6))


### Bug Fixes

* **docker:** missing write permissions on scripts ([ec4b41a](https://github.com/pocket-id/pocket-id/commit/ec4b41a1d26ea00bb4a95f654ac4cc745b2ce2e8))

## [](https://github.com/pocket-id/pocket-id/compare/v0.38.0...v) (2025-03-11)


### Features

* api key authentication ([#291](https://github.com/pocket-id/pocket-id/issues/291)) ([62915d8](https://github.com/pocket-id/pocket-id/commit/62915d863a4adc09cf467b75c414a045be43c2bb))


### Bug Fixes

* alternative login method link on mobile ([9ef2ddf](https://github.com/pocket-id/pocket-id/commit/9ef2ddf7963c6959992f3a5d6816840534e926e9))

## [](https://github.com/pocket-id/pocket-id/compare/v0.37.0...v) (2025-03-10)


### Features

* add env variable to disable update check ([31198fe](https://github.com/pocket-id/pocket-id/commit/31198feec2ae77dd6673c42b42002871ddd02d37))


### Bug Fixes

* redirection not correctly if signing in with email code ([e5ec264](https://github.com/pocket-id/pocket-id/commit/e5ec264bfd535752565bcc107099a9df5cb8aba7))
* typo in account settings ([#307](https://github.com/pocket-id/pocket-id/issues/307)) ([c822192](https://github.com/pocket-id/pocket-id/commit/c8221921245deb3008f655740d1a9460dcdab2fc))

## [](https://github.com/pocket-id/pocket-id/compare/v0.36.0...v) (2025-03-10)


### Features

* **account:** add ability to sign in with login code ([#271](https://github.com/pocket-id/pocket-id/issues/271)) ([eb1426e](https://github.com/pocket-id/pocket-id/commit/eb1426ed2684b5ddd185db247a8e082b28dfd014))
* increase default item count per page ([a9713cf](https://github.com/pocket-id/pocket-id/commit/a9713cf6a1e3c879dc773889b7983e51bbe3c45b))


### Bug Fixes

* add back setup page ([6a8dd84](https://github.com/pocket-id/pocket-id/commit/6a8dd84ca9396ff3369385af22f7e1f081bec2b2))
* add timeout to update check ([04efc36](https://github.com/pocket-id/pocket-id/commit/04efc3611568a0b0127b542b8cc252d9e783af46))
* make sorting consistent around tables ([8e344f1](https://github.com/pocket-id/pocket-id/commit/8e344f1151628581b637692a1de0e48e7235a22d))

## [](https://github.com/pocket-id/pocket-id/compare/v0.35.6...v) (2025-03-06)


### Features

* display groups on the account page ([#296](https://github.com/pocket-id/pocket-id/issues/296)) ([0f14a93](https://github.com/pocket-id/pocket-id/commit/0f14a93e1d6a723b0994ba475b04702646f04464))
* enable sd_notify support ([#277](https://github.com/pocket-id/pocket-id/issues/277)) ([91f254c](https://github.com/pocket-id/pocket-id/commit/91f254c7bb067646c42424c5c62ebcd90a0c8792))


### Bug Fixes

* default sorting on tables ([#299](https://github.com/pocket-id/pocket-id/issues/299)) ([ff34e3b](https://github.com/pocket-id/pocket-id/commit/ff34e3b925321c80e9d7d42d0fd50e397d198435))

## [](https://github.com/pocket-id/pocket-id/compare/v0.35.5...v) (2025-03-03)


### Bug Fixes

* support `LOGIN` authentication method for SMTP ([#292](https://github.com/pocket-id/pocket-id/issues/292)) ([2d733fc](https://github.com/pocket-id/pocket-id/commit/2d733fc79faefca23d54b22768029c3ba3427410))

## [](https://github.com/pocket-id/pocket-id/compare/v0.35.4...v) (2025-03-03)


### Bug Fixes

* profile picture orientation if image is rotated with EXIF ([1026ee4](https://github.com/pocket-id/pocket-id/commit/1026ee4f5b5c7fda78b65c94a5d0f899525defd1))

## [](https://github.com/pocket-id/pocket-id/compare/v0.35.3...v) (2025-03-01)


### Bug Fixes

* add `groups` scope and claim to well known endpoint ([4bafee4](https://github.com/pocket-id/pocket-id/commit/4bafee4f58f5a76898cf66d6192916d405eea389))
* profile picture of other user can't be updated ([#273](https://github.com/pocket-id/pocket-id/issues/273)) ([ef25f6b](https://github.com/pocket-id/pocket-id/commit/ef25f6b6b84b52f1310d366d40aa3769a6fe9bef))
* support POST for OIDC userinfo endpoint ([1652cc6](https://github.com/pocket-id/pocket-id/commit/1652cc65f3f966d018d81a1ae22abb5ff1b4c47b))

## [](https://github.com/pocket-id/pocket-id/compare/v0.35.2...v) (2025-02-25)


### Bug Fixes

* add option to manually select SMTP TLS method ([#268](https://github.com/pocket-id/pocket-id/issues/268)) ([01a9de0](https://github.com/pocket-id/pocket-id/commit/01a9de0b04512c62d0f223de33d711f93c49b9cc))
* **ldap:** sync error if LDAP user collides with an existing user ([fde951b](https://github.com/pocket-id/pocket-id/commit/fde951b543281fedf9f602abae26b50881e3d157))

## [](https://github.com/pocket-id/pocket-id/compare/v0.35.1...v) (2025-02-24)


### Bug Fixes

* delete profile picture if user gets deleted ([9a167d4](https://github.com/pocket-id/pocket-id/commit/9a167d4076872e5e3e5d78d2a66ef7203ca5261b))
* updating profile picture of other user updates own profile picture ([887c5e4](https://github.com/pocket-id/pocket-id/commit/887c5e462a50c8fb579ca6804f1a643d8af78fe8))

## [](https://github.com/pocket-id/pocket-id/compare/v0.35.0...v) (2025-02-22)


### Bug Fixes

* add validation that `PUBLIC_APP_URL` can't contain a path ([a6ae7ae](https://github.com/pocket-id/pocket-id/commit/a6ae7ae28713f7fc8018ae2aa7572986df3e1a5b))
* binary profile picture can't be imported from LDAP ([840a672](https://github.com/pocket-id/pocket-id/commit/840a672fc35ca8476caf86d7efaba9d54bce86aa))

## [](https://github.com/pocket-id/pocket-id/compare/v0.34.0...v) (2025-02-19)


### Features

* add ability to upload a profile picture ([#244](https://github.com/pocket-id/pocket-id/issues/244)) ([652ee6a](https://github.com/pocket-id/pocket-id/commit/652ee6ad5d6c46f0d35c955ff7bb9bdac6240ca6))


### Bug Fixes

* app config strings starting with a number are parsed incorrectly ([816c198](https://github.com/pocket-id/pocket-id/commit/816c198a42c189cb1f2d94885d2e3623e47e2848))
* emails do not get rendered correctly in Gmail ([dca9e7a](https://github.com/pocket-id/pocket-id/commit/dca9e7a11a3ba5d3b43a937f11cb9d16abad2db5))

## [](https://github.com/pocket-id/pocket-id/compare/v0.33.0...v) (2025-02-16)


### Features

* add LDAP group membership attribute ([#236](https://github.com/pocket-id/pocket-id/issues/236)) ([39b46e9](https://github.com/pocket-id/pocket-id/commit/39b46e99a9b930ea39cf640c3080530cfff5be6e))

## [](https://github.com/pocket-id/pocket-id/compare/v0.32.0...v) (2025-02-14)


### Features

* add end session endpoint ([#232](https://github.com/pocket-id/pocket-id/issues/232)) ([7550333](https://github.com/pocket-id/pocket-id/commit/7550333fe2ff6424f3168f63c5179d76767532fd))


### Bug Fixes

* alignment of OIDC client details ([c3980d3](https://github.com/pocket-id/pocket-id/commit/c3980d3d28a7158a4dc9369af41f185b891e485e))
* layout of OIDC client details page on mobile ([3de1301](https://github.com/pocket-id/pocket-id/commit/3de1301fa84b3ab4fff4242d827c7794d44910f2))
* show  "Sync Now" and "Test Email" button even if UI config is disabled ([4d0fff8](https://github.com/pocket-id/pocket-id/commit/4d0fff821e2245050ce631b4465969510466dfae))

## [](https://github.com/pocket-id/pocket-id/compare/v0.31.0...v) (2025-02-13)


### Features

* add ability to set custom Geolite DB URL ([2071d00](https://github.com/pocket-id/pocket-id/commit/2071d002fc5c3b5ff7a3fca6a5c99f5517196853))

## [](https://github.com/pocket-id/pocket-id/compare/v0.30.0...v) (2025-02-12)


### Features

* add ability to override the UI configuration with environment variables ([4e85842](https://github.com/pocket-id/pocket-id/commit/4e858420e9d9713e19f3b35c45c882403717f72f))
* add warning for only having one passkey configured ([#220](https://github.com/pocket-id/pocket-id/issues/220)) ([39e403d](https://github.com/pocket-id/pocket-id/commit/39e403d00f3870f9e960427653a1d9697da27a6f))
* display source in user and group table ([#225](https://github.com/pocket-id/pocket-id/issues/225)) ([9ed2adb](https://github.com/pocket-id/pocket-id/commit/9ed2adb0f8da13725fd9a4ef6a7798c377d13513))


### Bug Fixes

* user linking in ldap group sync ([#222](https://github.com/pocket-id/pocket-id/issues/222)) ([2d78349](https://github.com/pocket-id/pocket-id/commit/2d78349b381d7ca10f47d3c03cef685a576b1b49))

## [](https://github.com/pocket-id/pocket-id/compare/v0.29.0...v) (2025-02-08)


### Features

* add custom ldap search filters ([#216](https://github.com/pocket-id/pocket-id/issues/216)) ([626f87d](https://github.com/pocket-id/pocket-id/commit/626f87d59211f4129098b91dc1d020edb4aca692))
* update host configuration to allow external access ([#218](https://github.com/pocket-id/pocket-id/issues/218)) ([bea1158](https://github.com/pocket-id/pocket-id/commit/bea115866fd8e4b15d3281c422d2fb72312758b1))

## [](https://github.com/pocket-id/pocket-id/compare/v0.28.1...v) (2025-02-05)


### Features

* add JSON support in custom claims ([15cde6a](https://github.com/pocket-id/pocket-id/commit/15cde6ac66bc857ac28df545a37c1f4341977595))
* add option to disable Caddy in the Docker container ([e864d5d](https://github.com/pocket-id/pocket-id/commit/e864d5dcbff1ef28dc6bf120e4503093a308c5c8))

## [](https://github.com/stonith404/pocket-id/compare/v0.28.0...v) (2025-02-04)


### Bug Fixes

* don't return error page if version info fetching failed ([d06257e](https://github.com/stonith404/pocket-id/commit/d06257ec9b5e46e25e40c174b4bef02dca0a1ea3))

## [](https://github.com/stonith404/pocket-id/compare/v0.27.2...v) (2025-02-03)


### Features

* allow LDAP users and groups to be deleted if LDAP gets disabled ([9ab1787](https://github.com/stonith404/pocket-id/commit/9ab178712aa3cc71546a89226e67b7ba91245251))
* map allowed groups to OIDC clients ([#202](https://github.com/stonith404/pocket-id/issues/202)) ([13b02a0](https://github.com/stonith404/pocket-id/commit/13b02a072f20ce10e12fd8b897cbf42a908f3291))


### Bug Fixes

* **caddy:** trusted_proxies for IPv6 enabled hosts ([#189](https://github.com/stonith404/pocket-id/issues/189)) ([37a835b](https://github.com/stonith404/pocket-id/commit/37a835b44e308622f6862de494738dd2bfb58ef0))
* missing user service dependency ([61e71ad](https://github.com/stonith404/pocket-id/commit/61e71ad43b8f0f498133d3eb2381382e7bc642b9))
* non LDAP user group can't be updated after update ([ecd74b7](https://github.com/stonith404/pocket-id/commit/ecd74b794f1ffb7da05bce0046fb8d096b039409))
* use cursor pointer on clickable elements ([7798580](https://github.com/stonith404/pocket-id/commit/77985800ae9628104e03e7f2e803b7ed9eaaf4e0))

## [](https://github.com/stonith404/pocket-id/compare/v0.27.1...v) (2025-01-27)


### Bug Fixes

* smtp hello for tls connections ([#180](https://github.com/stonith404/pocket-id/issues/180)) ([781ff7a](https://github.com/stonith404/pocket-id/commit/781ff7ae7b84b13892e7a565b7a78f20c52ee2c9))

## [](https://github.com/stonith404/pocket-id/compare/v0.27.0...v) (2025-01-24)


### Bug Fixes

* add `__HOST` prefix to cookies ([#175](https://github.com/stonith404/pocket-id/issues/175)) ([164ce6a](https://github.com/stonith404/pocket-id/commit/164ce6a3d7fa8ae5275c94302952cf318e3b3113))
* send hostname derived from `PUBLIC_APP_URL` with SMTP EHLO command ([397544c](https://github.com/stonith404/pocket-id/commit/397544c0f3f2b49f1f34ae53e6b9daf194d1ae28))
* use OS hostname for SMTP EHLO message ([47c39f6](https://github.com/stonith404/pocket-id/commit/47c39f6d382c496cb964262adcf76cc8dbb96da3))

## [](https://github.com/stonith404/pocket-id/compare/v0.26.0...v) (2025-01-22)


### Features

* display private IP ranges correctly in audit log ([#139](https://github.com/stonith404/pocket-id/issues/139)) ([72923bb](https://github.com/stonith404/pocket-id/commit/72923bb86dc5d07d56aea98cf03320667944b553))


### Bug Fixes

* add save changes dialog before sending test email ([#165](https://github.com/stonith404/pocket-id/issues/165)) ([d02f475](https://github.com/stonith404/pocket-id/commit/d02f4753f3fbda75cd415ebbfe0702765c38c144))
* ensure the downloaded GeoLite2 DB is not corrupted & prevent RW race condition ([#138](https://github.com/stonith404/pocket-id/issues/138)) ([f7710f2](https://github.com/stonith404/pocket-id/commit/f7710f298898d322885c1c83680e26faaa0bb800))

## [](https://github.com/stonith404/pocket-id/compare/v0.25.1...v) (2025-01-20)


### Features

* support wildcard callback URLs ([8a1db0c](https://github.com/stonith404/pocket-id/commit/8a1db0cb4a5d4b32b4fdc19d41fff688a7c71a56))


### Bug Fixes

* non LDAP users get created with a empty LDAP ID string ([3f02d08](https://github.com/stonith404/pocket-id/commit/3f02d081098ad2caaa60a56eea4705639f80d01f))

## [](https://github.com/stonith404/pocket-id/compare/v0.25.0...v) (2025-01-19)


### Bug Fixes

* disable account details inputs if user is imported from LDAP ([a8b9d60](https://github.com/stonith404/pocket-id/commit/a8b9d60a86e80c10d6fba07072b1d32cec400ecb))

## [](https://github.com/stonith404/pocket-id/compare/v0.24.1...v) (2025-01-19)


### Features

* add LDAP sync ([#106](https://github.com/stonith404/pocket-id/issues/106)) ([5101b14](https://github.com/stonith404/pocket-id/commit/5101b14eec68a9507e1730994178d0ebe8185876))
* allow sign in with email ([#100](https://github.com/stonith404/pocket-id/issues/100)) ([06b90ed](https://github.com/stonith404/pocket-id/commit/06b90eddd645cce57813f2536e4a6a8836548f2b))
* automatically authorize client if signed in ([d5dd118](https://github.com/stonith404/pocket-id/commit/d5dd118a3f4ad6eed9ca496c458201bb10f148a0))


### Bug Fixes

* always set secure on cookie ([#130](https://github.com/stonith404/pocket-id/issues/130)) ([fda08ac](https://github.com/stonith404/pocket-id/commit/fda08ac1cd88842e25dc47395ed1288a5cfac4f8))
* don't panic if LDAP sync fails on startup ([e284e35](https://github.com/stonith404/pocket-id/commit/e284e352e2b95fac1d098de3d404e8531de4b869))
* improve spacing of checkboxes on application configuration page ([090eca2](https://github.com/stonith404/pocket-id/commit/090eca202d198852e6fbf4e6bebaf3b5ada13944))
* search input not displayed if response hasn't any items ([05a98eb](https://github.com/stonith404/pocket-id/commit/05a98ebe87d7a88e8b96b144c53250a40d724ec3))
* session duration ignored in cookie expiration ([bc8f454](https://github.com/stonith404/pocket-id/commit/bc8f454ea173ecc60e06450a1d22e24207f76714))

## [](https://github.com/stonith404/pocket-id/compare/v0.24.0...v) (2025-01-13)


### Bug Fixes

* audit log table overflow if row data is long ([4d337a2](https://github.com/stonith404/pocket-id/commit/4d337a20c5cb92ef80bb7402f9b99b08e3ad0b6b))
* optional arguments not working with `create-one-time-access-token.sh` ([8885571](https://github.com/stonith404/pocket-id/commit/888557171d61589211b10f70dce405126216ad61))
* remove restrictive validation for group names ([be6e25a](https://github.com/stonith404/pocket-id/commit/be6e25a167de8bf07075b46f09d9fc1fa6c74426))

## [](https://github.com/stonith404/pocket-id/compare/v0.23.0...v) (2025-01-11)


### Features

* add sorting for tables ([fd69830](https://github.com/stonith404/pocket-id/commit/fd69830c2681985e4fd3c5336a2b75c9fb7bc5d4))


### Bug Fixes

* pkce state not correctly reflected in oidc client info ([61d18a9](https://github.com/stonith404/pocket-id/commit/61d18a9d1b167ff59a59523ff00d00ca8f23258d))
* send test email to the user that has requested it ([a649c4b](https://github.com/stonith404/pocket-id/commit/a649c4b4a543286123f4d1f3c411fe1a7e2c6d71))

## [](https://github.com/stonith404/pocket-id/compare/v0.22.0...v) (2025-01-03)


### Features

* add PKCE for non public clients ([adcf3dd](https://github.com/stonith404/pocket-id/commit/adcf3ddc6682794e136a454ef9e69ddd130626a8))
* use same table component for OIDC client list as all other lists ([2d31fc2](https://github.com/stonith404/pocket-id/commit/2d31fc2cc9201bb93d296faae622f52c6dcdfebc))

## [](https://github.com/stonith404/pocket-id/compare/v0.21.0...v) (2025-01-01)


### Features

* add warning if passkeys missing ([2d0bd8d](https://github.com/stonith404/pocket-id/commit/2d0bd8dcbfb73650b7829cb66f40decb284bd73b))


### Bug Fixes

* allow first and last name of user to be between 1 and 50 characters ([1ff20ca](https://github.com/stonith404/pocket-id/commit/1ff20caa3ccd651f9fb30f958ffb807dfbbcbd8a))
* hash in callback url is incorrectly appended ([f6f2736](https://github.com/stonith404/pocket-id/commit/f6f2736bba65eee017f2d8cdaa70621574092869))
* make user validation consistent between pages ([333a1a1](https://github.com/stonith404/pocket-id/commit/333a1a18d59f675111f4ed106fa5614ef563c6f4))
* passkey can't be added if `PUBLIC_APP_URL` includes a port ([0729ce9](https://github.com/stonith404/pocket-id/commit/0729ce9e1a8dab9912900a01dcd0fbd892718a1a))

## [](https://github.com/stonith404/pocket-id/compare/v0.20.1...v) (2024-12-17)


### Features

* improve error state design for login page ([0716c38](https://github.com/stonith404/pocket-id/commit/0716c38fb8ce7fa719c7fe0df750bdb213786c21))


### Bug Fixes

* OIDC client logo gets removed if other properties get updated ([789d939](https://github.com/stonith404/pocket-id/commit/789d9394a533831e7e2fb8dc3f6b338787336ad8))

## [](https://github.com/stonith404/pocket-id/compare/v0.20.0...v) (2024-12-13)


### Bug Fixes

* `create-one-time-access-token.sh` script not compatible with postgres ([34e3519](https://github.com/stonith404/pocket-id/commit/34e35193f9f3813f6248e60f15080d753e8da7ae))
* wrong date time datatype used for read operations with Postgres ([bad901e](https://github.com/stonith404/pocket-id/commit/bad901ea2b661aadd286e5e4bed317e73bd8a70d))

## [](https://github.com/stonith404/pocket-id/compare/v0.19.0...v) (2024-12-12)


### Features

* add support for Postgres database provider ([#79](https://github.com/stonith404/pocket-id/issues/79)) ([9d20a98](https://github.com/stonith404/pocket-id/commit/9d20a98dbbc322fa6f0644e8b31e6b97769887ce))

## [](https://github.com/stonith404/pocket-id/compare/v0.18.0...v) (2024-11-29)


### Features

* **geolite:** add Tailscale IP detection with CGNAT range check  ([#77](https://github.com/stonith404/pocket-id/issues/77)) ([edce3d3](https://github.com/stonith404/pocket-id/commit/edce3d337129c9c6e8a60df2122745984ba0f3e0))

## [](https://github.com/stonith404/pocket-id/compare/v0.17.0...v) (2024-11-28)


### Features

* add option to disable TLS for email sending ([f9fa2c6](https://github.com/stonith404/pocket-id/commit/f9fa2c6706a8bf949fe5efd6664dec8c80e18659))
* allow empty user and password in SMTP configuration ([a9f4dad](https://github.com/stonith404/pocket-id/commit/a9f4dada321841d3611b15775307228b34e7793f))


### Bug Fixes

* email save toast shows two times ([f2bfc73](https://github.com/stonith404/pocket-id/commit/f2bfc731585ad7424eb8c4c41c18368fc0f75ffc))

## [](https://github.com/stonith404/pocket-id/compare/v0.16.0...v) (2024-11-26)


### ⚠ BREAKING CHANGES

* add option to specify the Max Mind license key for the Geolite2 db

### Features

* add option to specify the Max Mind license key for the Geolite2 db ([fcf08a4](https://github.com/stonith404/pocket-id/commit/fcf08a4d898160426442bd80830f4431988f4313))


### Bug Fixes

* don't try to create a new user if the Docker user is not root ([#71](https://github.com/stonith404/pocket-id/issues/71)) ([0e95e9c](https://github.com/stonith404/pocket-id/commit/0e95e9c56f4c3f84982f508fdb6894ba747952b4))

## [](https://github.com/stonith404/pocket-id/compare/v0.15.0...v) (2024-11-24)


### Features

* add health check ([058084e](https://github.com/stonith404/pocket-id/commit/058084ed64816b12108e25bf04af988fc97772ed))
* improve error message for invalid callback url ([f637a89](https://github.com/stonith404/pocket-id/commit/f637a89f579aefb8dc3c3c16a27ef0bc453dfe40))

## [](https://github.com/stonith404/pocket-id/compare/v0.14.0...v) (2024-11-21)


### Features

* add option to skip TLS certificate check and ability to send test email ([653d948](https://github.com/stonith404/pocket-id/commit/653d948f73b61e6d1fd3484398fef1a2a37e6d92))
* add PKCE support ([3613ac2](https://github.com/stonith404/pocket-id/commit/3613ac261cf65a2db0620ff16dc6df239f6e5ecd))


### Bug Fixes

* mobile layout overflow on application configuration page ([e784093](https://github.com/stonith404/pocket-id/commit/e784093342f9977ea08cac65ff0c3de4d2644872))

## [](https://github.com/stonith404/pocket-id/compare/v0.13.1...v) (2024-11-11)


### Features

* add audit log event for one time access token sign in ([aca2240](https://github.com/stonith404/pocket-id/commit/aca2240a50a12e849cfb6e1aa56390b000aebae0))


### Bug Fixes

* overflow of pagination control on mobile ([de45398](https://github.com/stonith404/pocket-id/commit/de4539890349153c467013c24c4d6b30feb8fed8))
* time displayed incorrectly in audit log ([3d3fb4d](https://github.com/stonith404/pocket-id/commit/3d3fb4d855ef510f2292e98fcaaaf83debb5d3e0))

## [](https://github.com/stonith404/pocket-id/compare/v0.13.0...v) (2024-11-01)


### Features

* add list empty indicator ([becfc00](https://github.com/stonith404/pocket-id/commit/becfc0004a87c01e18eb92ac85bf4e33f105b6a3))


### Bug Fixes

* errors in middleware do not abort the request ([376d747](https://github.com/stonith404/pocket-id/commit/376d747616b1e835f252d20832c5ae42b8b0b737))
* typo in Self-Account Editing description ([5b9f4d7](https://github.com/stonith404/pocket-id/commit/5b9f4d732615f428c13d3317da96a86c5daebd89))

## [](https://github.com/stonith404/pocket-id/compare/v0.12.0...v) (2024-10-31)


### Features

* add ability to define expiration of one time link ([2ccabf8](https://github.com/stonith404/pocket-id/commit/2ccabf835c2c923d6986d9cafb4e878f5110b91a))

## [](https://github.com/stonith404/pocket-id/compare/v0.11.0...v) (2024-10-28)


### Features

* add option to disable self-account editing ([8304065](https://github.com/stonith404/pocket-id/commit/83040656525cf7b6c8f2acf416c5f8f3288f3d48))
* add validation to custom claim input ([7bfc3f4](https://github.com/stonith404/pocket-id/commit/7bfc3f43a591287c038187ed5e782de6b9dd738b))
* custom claims ([#53](https://github.com/stonith404/pocket-id/issues/53)) ([c056089](https://github.com/stonith404/pocket-id/commit/c056089c6043a825aaaaecf0c57454892a108f1d))

## [](https://github.com/stonith404/pocket-id/compare/v0.10.0...v) (2024-10-25)


### Features

* add `email_verified` claim ([5565f60](https://github.com/stonith404/pocket-id/commit/5565f60d6d62ca24bedea337e21effc13e5853a5))


### Bug Fixes

* powered by link text color in light mode ([18c5103](https://github.com/stonith404/pocket-id/commit/18c5103c20ce79abdc0f724cdedd642c09269e78))

## [](https://github.com/stonith404/pocket-id/compare/v0.9.0...v) (2024-10-23)


### Features

* add script for creating one time access token ([a1985ce](https://github.com/stonith404/pocket-id/commit/a1985ce1b200550e91c5cb42a8d19899dcec831e))
* add version information to footer and update link if new update is available ([70ad0b4](https://github.com/stonith404/pocket-id/commit/70ad0b4f39699fd81ffdfd5c8d6839f49348be78))


### Bug Fixes

* cache version information for 3 hours ([29d632c](https://github.com/stonith404/pocket-id/commit/29d632c1514d6edacdfebe6deae4c95fc5a0f621))
* improve text for initial admin account setup ([0a07344](https://github.com/stonith404/pocket-id/commit/0a0734413943b1fff27d8f4ccf07587e207e2189))
* increase callback url count ([f3f0e1d](https://github.com/stonith404/pocket-id/commit/f3f0e1d56d7656bdabbd745a4eaf967f63193b6c))
* no DTO was returned from exchange one time access token endpoint ([824c5cb](https://github.com/stonith404/pocket-id/commit/824c5cb4f3d6be7f940c1758112fbe9322df5768))

## [](https://github.com/stonith404/pocket-id/compare/v0.8.1...v) (2024-10-18)


### Features

* add environment variable to change the caddy port in Docker ([ff06bf0](https://github.com/stonith404/pocket-id/commit/ff06bf0b34496ce472ba6d3ebd4ea249f21c0ec3))
* use improve table for users and audit logs ([11ed661](https://github.com/stonith404/pocket-id/commit/11ed661f86a512f78f66d604a10c1d47d39f2c39))


### Bug Fixes

* allow copy to clipboard for client secret ([29748cc](https://github.com/stonith404/pocket-id/commit/29748cc6c7b7e5a6b54bfe837e0b1a98fa1ad594))

## [](https://github.com/stonith404/pocket-id/compare/v0.8.0...v) (2024-10-11)


### Bug Fixes

* add key id to JWK ([282ff82](https://github.com/stonith404/pocket-id/commit/282ff82b0c7e2414b3528c8ca325758245b8ae61))

## [](https://github.com/stonith404/pocket-id/compare/v0.7.1...v) (2024-10-04)


### Features

* add location based on ip to the audit log ([025378d](https://github.com/stonith404/pocket-id/commit/025378d14edd2d72da76e90799a0ccdd42cf672c))

## [](https://github.com/stonith404/pocket-id/compare/v0.7.0...v) (2024-10-03)


### Bug Fixes

* initials don't get displayed if Gravatar avatar doesn't exist ([e095628](https://github.com/stonith404/pocket-id/commit/e09562824a794bc7d240e9d229709d4b389db7d5))

## [](https://github.com/stonith404/pocket-id/compare/v0.6.0...v) (2024-10-03)


### ⚠ BREAKING CHANGES

* add ability to set light and dark mode logo

### Features

* add ability to set light and dark mode logo ([be45eed](https://github.com/stonith404/pocket-id/commit/be45eed125e33e9930572660a034d5f12dc310ce))

## [](https://github.com/stonith404/pocket-id/compare/v0.5.3...v) (2024-10-02)


### Features

* add copy to clipboard option for OIDC client information ([f82020c](https://github.com/stonith404/pocket-id/commit/f82020ccfb0d4fbaa1dd98182188149d8085252a))
* add gravatar profile picture integration ([365734e](https://github.com/stonith404/pocket-id/commit/365734ec5d8966c2ab877c60cfb176b9cdc36880))
* add user groups ([24c948e](https://github.com/stonith404/pocket-id/commit/24c948e6a66f283866f6c8369c16fa6cbcfa626c))


### Bug Fixes

* only return user groups if it is explicitly requested ([a4a90a1](https://github.com/stonith404/pocket-id/commit/a4a90a16a9726569a22e42560184319b25fd7ca6))

## [](https://github.com/stonith404/pocket-id/compare/v0.5.2...v) (2024-09-26)


### Bug Fixes

* add space to "Firstname" and "Lastname" label ([#31](https://github.com/stonith404/pocket-id/issues/31)) ([d6a9bb4](https://github.com/stonith404/pocket-id/commit/d6a9bb4c09efb8102da172e49c36c070b341f0fc))
* port environment variables get ignored in caddyfile ([3c67765](https://github.com/stonith404/pocket-id/commit/3c67765992d7369a79812bc8cd216c9ba12fd96e))

## [](https://github.com/stonith404/pocket-id/compare/v0.5.1...v) (2024-09-19)


### Bug Fixes

* updated application name doesn't apply to webauthn credential ([924bb14](https://github.com/stonith404/pocket-id/commit/924bb1468bbd8e42fa6a530ef740be73ce3b3914))

## [](https://github.com/stonith404/pocket-id/compare/v0.5.0...v) (2024-09-16)


### Features

* **email:** improve email templating ([#27](https://github.com/stonith404/pocket-id/issues/27)) ([64cf562](https://github.com/stonith404/pocket-id/commit/64cf56276a07169bc601a11be905c1eea67c4750))


### Bug Fixes

* debounce oidc client and user search ([9c2848d](https://github.com/stonith404/pocket-id/commit/9c2848db1d93c230afc6c5f64e498e9f6df8c8a7))

## [](https://github.com/stonith404/pocket-id/compare/v0.4.1...v) (2024-09-09)


### Features

* add audit log with email notification ([#26](https://github.com/stonith404/pocket-id/issues/26)) ([9121239](https://github.com/stonith404/pocket-id/commit/9121239dd7c14a2107a984f9f94f54227489a63a))

## [](https://github.com/stonith404/pocket-id/compare/v0.4.0...v) (2024-09-06)


### Features

* add name claim to userinfo endpoint and id token ([4e7574a](https://github.com/stonith404/pocket-id/commit/4e7574a297307395603267c7a3285d538d4111d8))


### Bug Fixes

* limit width of content on large screens ([c6f83a5](https://github.com/stonith404/pocket-id/commit/c6f83a581ad385391d77fec7eeb385060742f097))
* show error message if error occurs while authorizing new client ([8038a11](https://github.com/stonith404/pocket-id/commit/8038a111dd7fa8f5d421b29c3bc0c11d865dc71b))

## [](https://github.com/stonith404/pocket-id/compare/v0.3.1...v) (2024-09-03)


### Features

* add setup details to oidc client details ([fd21ce5](https://github.com/stonith404/pocket-id/commit/fd21ce5aac1daeba04e4e7399a0720338ea710c2))
* add support for more username formats ([903b0b3](https://github.com/stonith404/pocket-id/commit/903b0b39181c208e9411ee61849d2671e7c56dc5))


### Bug Fixes

* non pointer passed to create user ([e7861df](https://github.com/stonith404/pocket-id/commit/e7861df95a6beecab359d1c56f4383373f74bb73))
* oidc client logo not displayed on authorize page ([28ed064](https://github.com/stonith404/pocket-id/commit/28ed064668afeec8f80adda59ba94f1fc2fbce17))
* typo in hasLogo property of oidc dto ([2b9413c](https://github.com/stonith404/pocket-id/commit/2b9413c7575e1322f8547490a9b02a1836bad549))

## [](https://github.com/stonith404/pocket-id/compare/v0.3.0...v) (2024-08-24)


### Bug Fixes

* empty lists don't get returned correctly from the api ([97f7fc4](https://github.com/stonith404/pocket-id/commit/97f7fc4e288c2bb49210072a7a151b58ef44f5b5))

## [](https://github.com/stonith404/pocket-id/compare/v0.2.1...v) (2024-08-23)


### Features

* add support for multiple callback urls ([8166e2e](https://github.com/stonith404/pocket-id/commit/8166e2ead7fc71a0b7a45950b05c5c65a60833b6))


### Bug Fixes

* db migration for multiple callback urls ([552d7cc](https://github.com/stonith404/pocket-id/commit/552d7ccfa58d7922ecb94bdfe6a86651b4cf2745))

## [](https://github.com/stonith404/pocket-id/compare/v0.2.0...v) (2024-08-19)


### Bug Fixes

* session duration can't be updated ([4780548](https://github.com/stonith404/pocket-id/commit/478054884389ed8a08d707fd82da7b31177a67e5))

## [](https://github.com/stonith404/pocket-id/compare/v0.1.3...v) (2024-08-19)


### Features

* add `INTERNAL_BACKEND_URL` env variable ([0595d73](https://github.com/stonith404/pocket-id/commit/0595d73ea5afbd7937b8f292ffe624139f818f41))
* add user info endpoint to support more oidc clients ([fdc1921](https://github.com/stonith404/pocket-id/commit/fdc1921f5dcb5ac6beef8d1c9b1b7c53f514cce5))
* change default logo ([9eec7a3](https://github.com/stonith404/pocket-id/commit/9eec7a3e9eb7f690099f38a5d4cf7c2516ea9ef9))

## [](https://github.com/stonith404/pocket-id/compare/v0.1.2...v) (2024-08-13)


### Bug Fixes

* add missing passkey flags to make icloud passkeys work ([cc407e1](https://github.com/stonith404/pocket-id/commit/cc407e17d409041ed88b959ce13bd581663d55c3))
* logo not white in dark mode ([5749d05](https://github.com/stonith404/pocket-id/commit/5749d0532fc38bf2fc66571878b7c71643895c9e))

## [](https://github.com/stonith404/pocket-id/compare/v0.1.1...v) (2024-08-13)


### Features

* add option to change session duration ([475b932](https://github.com/stonith404/pocket-id/commit/475b932f9d0ec029ada844072e9d89bebd4e902c))


### Bug Fixes

* a non admin user was able to make himself an admin ([df0cd38](https://github.com/stonith404/pocket-id/commit/df0cd38deeea516c47b26a080eed522f19f7290f))
* background image not loading ([7b44189](https://github.com/stonith404/pocket-id/commit/7b4418958ebfffffd216ef5ba7313cfaad9bc9fa))
* background image on mobile ([4a808c8](https://github.com/stonith404/pocket-id/commit/4a808c86ac204f9b58cfa02f5ceb064162a87076))
* disable search engine indexing ([8395492](https://github.com/stonith404/pocket-id/commit/83954926f5ee328ebf75a75bb47b380ec0680378))

## [](https://github.com/stonith404/pocket-id/compare/v0.1.0...v) (2024-08-12)


### Features

* add rounded corners to logo ([bec908f](https://github.com/stonith404/pocket-id/commit/bec908f9078aaa4eec03b730fc36b9fffb1ece74))


### Bug Fixes

* one time link not displayed correctly ([486771f](https://github.com/stonith404/pocket-id/commit/486771f433872d08164156d5d6fb0aeb5ae0d125))

##  (2024-08-12)

