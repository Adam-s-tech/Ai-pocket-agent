# Releasing Pocket Agent

Publishing a GitHub release with a tag like `v1.0.0` runs
`.github/workflows/ci.yml`. It tests, builds the on-device engine and every
package, and attaches them to the release. The workflow is written by hand
(not by `giode ci`) because the engine is built before the apps, but its
secrets have the names `giode ci`'s workflows use.

Use three-part tags (`v1.2.0`). The version is the tag and the build number
(Android `versionCode`, iOS `CFBundleVersion`) is the workflow's run number,
which grows with every release, as each store upload needs. A four-part tag
(`v1.2.0.7`) sets the build number itself; it must still be higher than the
last one uploaded.

| Where | What the release gets | Needs |
| --- | --- | --- |
| GitHub, Android | `pocket-agent.apk` (arm64, Android 9+), which downloads the engine | nothing; the release keystore so updates install over earlier versions |
| GitHub, engine | `libpocketllama-*` and `engine-manifest.json`, which GitHub APKs download | nothing |
| Google Play | `pocket-agent.aab` with the engine packaged in, and `native-debug-symbols.zip` | a Play Console account and the release keystore |
| App Store (iPhone, iPad) | `PocketAgent.ipa` with the engine compiled in | an Apple Developer account, the distribution certificate and profiles |
| (no iOS signing) | `PocketAgent-simulator.zip` | nothing |

Pocket Agent is a phone and tablet app (`app.platforms = ["android", "ios"]`):
there are no Microsoft Store or Mac App Store packages, and no desktop
downloads.

**Why the Play bundle differs from the APK.** Google Play doesn't let apps
download native code, and the GitHub APK downloads the llama.cpp engine. The
bundle is built with `-tags play` (`make aab`): `scripts/engine-aar.sh` packs
both engine builds (armv8.2 and armv8) into the app, which loads the one for
the phone's CPU and never downloads or updates the engine. A new engine
reaches Play users with the next app update.

## Repository secrets

Set these in GitHub under Settings → Secrets and variables → Actions. Each
`*_BASE64` secret is the file encoded with `base64 -i file | pbcopy` on macOS,
or `base64 -w0 file` on Linux. Without a secret, that step is skipped and the
rest of the release still builds.

| Secret | What it is |
| --- | --- |
| `ANDROID_KEYSTORE_BASE64`, `GIODE_KEYSTORE_PASS` | The release keystore made by `giode keystore`, and its password: signs the APK and the bundle |
| `IOS_CERT_BASE64`, `IOS_CERT_PASS` | The "Apple Distribution" certificate as a `.p12`, and its password |
| `IOS_PROFILE_BASE64` | The App Store provisioning profile for `com.mehran9675.pocketagent` |
| `IOS_SHARE_PROFILE_BASE64` | The App Store provisioning profile for the share extension, `com.mehran9675.pocketagent.share`; without it the app is built without the share extension |
| `IOS_TEAM_ID` | Optional: your Apple team ID, when the profile doesn't make it clear |

Earlier versions of the workflow used other names. Rename them:
`ANDROID_KEYSTORE_PASS` → `GIODE_KEYSTORE_PASS`, `APPLE_CERTIFICATE_BASE64` →
`IOS_CERT_BASE64`, `APPLE_CERTIFICATE_PASSWORD` → `IOS_CERT_PASS`,
`APPLE_PROVISION_PROFILE_BASE64` → `IOS_PROFILE_BASE64`,
`APPLE_SHARE_PROFILE_BASE64` → `IOS_SHARE_PROFILE_BASE64`. `KEYCHAIN_PASSWORD`
and `APPLE_APPID` are no longer used: store builds always use `app.id`.

## Google Play

1. Open a Google Play Console developer account.
2. Make the upload keystore once, and keep it and its password safe:

   ```sh
   GIODE_KEYSTORE_PASS=... giode keystore
   ```

   It writes `release.keystore` (gitignored), which `android.keystore` in
   `giode.toml` already names: `giode build -release` fails without that
   setting. Add the file as `ANDROID_KEYSTORE_BASE64` and the password as
   `GIODE_KEYSTORE_PASS`.
3. In Play Console, create the app "Pocket Agent" (app, free). The package
   name is `com.mehran9675.pocketagent` (`app.id`), fixed once the first
   bundle is uploaded.
4. Publish a release on GitHub, download `pocket-agent.aab` and upload it to
   an internal testing track first. Accept **Play App Signing**: Google keeps
   the app signing key and your keystore becomes the upload key. Upload
   `native-debug-symbols.zip` to the same release (App bundle explorer →
   Downloads → Native debug symbols) so crash reports are readable.
5. Fill in App content and the store listing (below), then promote to
   production.

Notes:

- **Target API.** `android.target_sdk = 36`: Google Play requires Android 16
  (API 36) for new apps and updates since 31 August 2026 (the CLI defaults to
  35, which Play now rejects). The workflow installs `platforms;android-36`.
- **Back gesture.** Android 16 stops calling `onBackPressed()`, which Gio's
  activity uses, for apps targeting API 36 (predictive back), so the back
  gesture would close the app. `android.application_attributes` opts out
  (`enableOnBackInvokedCallback = "false"`). Test the back gesture on an
  Android 16 device (closing a sheet, the drawer, going back a page) before
  each Play release.
- **16 KB pages.** Play requires native libraries aligned for 16 KB pages.
  The app's own library is; the engine is built with
  `ANDROID_SUPPORT_FLEXIBLE_PAGE_SIZES=ON` for that. Check the bundle in App
  bundle explorer after the first upload.
- **Foreground service.** The app declares a `dataSync` foreground service so
  an agent run or a reply finishes when the app leaves the screen. Play
  Console → App content → Foreground service permissions asks for it: choose
  "Data sync" and describe it as "finishes a chat reply or an agent task the
  user started (network requests to the AI provider) after the user leaves
  the app"; it asks for a short video of that.
- **APK and Play installs don't mix.** With Play App Signing, Play installs
  are signed with Google's key and GitHub APKs with yours: switching between
  them needs an uninstall (which deletes chats and models).

## App Store (iPhone and iPad)

1. Join the Apple Developer Program.
2. In Certificates, Identifiers & Profiles:
   - Register the App ID `com.mehran9675.pocketagent` (the `app.id` in
     `giode.toml`) with the App Groups capability.
   - Register the App Group `group.com.mehran9675.pocketagent`.
   - Register the share extension's App ID
     `com.mehran9675.pocketagent.share`, also with App Groups, and enable the
     group on both App IDs.
   - Create an "Apple Distribution" certificate; export it with its key as a
     `.p12` (`IOS_CERT_BASE64`, `IOS_CERT_PASS`).
   - Create an "App Store Connect" (distribution) provisioning profile for each
     App ID (`IOS_PROFILE_BASE64`, `IOS_SHARE_PROFILE_BASE64`).
3. In App Store Connect, create the app: platform iOS, name "Pocket Agent",
   bundle ID `com.mehran9675.pocketagent`, a SKU such as `pocketagent`.
4. Add the secrets and publish a release. Upload `PocketAgent.ipa` with the
   Transporter app (or `xcrun altool --upload-app -t ios -f PocketAgent.ipa`).
   It runs on iPhone and iPad (iOS 15+) and has llama.cpp built in.
5. Fill in the listing (below), App Privacy, the age rating and the review
   notes, then submit.

Already in the build: the usage texts for the camera, microphone and speech
recognition, `ITSAppUsesNonExemptEncryption = false` and the privacy manifest
(`[privacy]`, no tracking, no data collected). The icon is
`assets/appicon.png`, opaque and full-bleed.

## Store listing

**Name:** Pocket Agent

**Apple subtitle (≤ 30):** Private AI chat and agents

**Play short description (≤ 80):** Private AI chat, research and agents: on-device models or your own API keys.

**Full description:**

> Pocket Agent is a private AI workspace for your phone and tablet. Chat with
> models that run on the device itself, or with the cloud models you choose
> using your own API keys. Hand multi-step tasks to agents, and run deep
> research that reads dozens of sources and writes a cited report.
>
> ON-DEVICE MODELS
> llama.cpp runs GGUF models on the device (Metal on iPhone and iPad). Pick
> from a catalog sized for phones - Qwen3, Llama 3.2, Gemma 3, Phi-4 mini,
> SmolLM2 - with a fit rating for your device's memory, resumable downloads,
> Hugging Face search and GGUF import. Every llama.cpp sampler and cache
> setting is there when you want it. On-device chats never leave the device.
>
> CLOUD MODELS WITH YOUR KEYS
> Anthropic, OpenAI, Google Gemini, xAI, Mistral, DeepSeek, Moonshot, Z.ai,
> Alibaba Cloud, MiniMax, Perplexity and Cohere; OpenRouter, Groq, Together,
> Fireworks, Cerebras and other inference platforms; any OpenAI-compatible
> endpoint; or Ollama, LM Studio, llama.cpp server and vLLM on your own
> computer. Your messages go straight to the provider you pick.
>
> AGENTS AND DEEP RESEARCH
> Built-in agents and your own keep a live plan, use tools and write files to
> a workspace you can open and share. Deep research plans sub-questions, reads
> sources in parallel and streams a report with inline citations.
>
> TOOLS YOU CONTROL
> Web search, page reading, a calculator, a JavaScript sandbox, long-term
> memory, MCP servers and SKILL.md skills - each tool asks first: allow once,
> always allow or deny.
>
> MADE FOR TOUCH
> Voice input, photos and documents (PDF, Word, Excel, PowerPoint and more)
> read on the device, streaming Markdown with visible thinking, an SSH
> terminal, notifications when a background reply finishes, 25 languages,
> dark and light themes.
>
> PRIVATE BY DESIGN
> No account, no ads, no analytics. Conversations stay in the app's storage
> and API keys in the system's secure storage.

The terminal is SSH on both; on Android it also has a local shell, so the Play
text may add "and a local shell".

**Apple keywords (≤ 100):** `ai,chat,llm,agent,research,local,offline,llama,gguf,openai,claude,gemini,ollama,assistant`

**Category:** Productivity (both stores). Apple secondary: Developer Tools.

**Website / support URL:** https://xmhrnx.github.io/pocket-agent/

**Privacy policy URL:** https://xmhrnx.github.io/pocket-agent/#privacy

**Screenshots.** Take them with `make dev` (the offline Demo provider and the
MCP server, see docs/building.md) at the stores' sizes, from the same screens
as `docs/screenshots/`: new chat, a Markdown reply, the model picker, the
conversation drawer; on iPad an agent run with the inspector and deep
research. The files in `docs/screenshots` are 1x (400×860 and 1366×1024) and
too small: the App Store wants 1290×2796 (iPhone 6.9") and 2048×2732 or
2732×2048 (iPad 13"); Google Play wants phone shots no longer than twice their
width (for example 1080×1920), plus a 512×512 icon (`assets/appicon.png`
scaled) and a 1024×500 feature graphic.

### App Privacy (Apple) and Data safety (Google)

- **Apple App Privacy:** "Data Not Collected". Nothing reaches the developer:
  chats and attachments go only to the AI provider the user sets up with their
  own key (or nowhere, on device), web searches to the search engine they
  pick, and everything else stays on the device. No tracking.
- **Google Data safety:** no data collected, no data shared. Data is
  encrypted in transit (HTTPS, SSH). There is no account, so no deletion
  request: uninstalling deletes everything. If a reviewer counts messages sent
  to the user-chosen AI provider as collection, declare "Messages - Other
  in-app messages" and "Photos" and "Files and docs" as collected, not shared,
  for App functionality, optional, processed by a service the user chose.

### Age rating

- **Apple:** answer Yes to "Unrestricted Web Access" (the model reads any web
  page and shows images from it) and declare the AI-generated content, whose
  replies come unfiltered from third-party models. Expect the highest age
  rating (18+).
- **Google (IARC):** no violence, sex or gambling content of its own; answer
  that users can reach unrestricted web content and AI-generated text. Target
  audience: 18 and over.

### Export compliance

Answered in the build: `ITSAppUsesNonExemptEncryption = false`. The app uses
HTTPS (the system's and Go's TLS) and SSH, standard algorithms only, and is
offered everywhere through the store, as EasySend and GoCode are.

## Known store limitations and review risks

- **Engine on Play:** packaged, not downloaded (see above). The engine on
  Play changes only with app updates.
- **Local shell (Android):** the terminal and the agents' device-shell tool run
  the system's `sh`. Google Play allows terminal apps, but a reviewer may ask;
  nothing executable is downloaded. iOS has no local shell.
- **JavaScript sandbox and skills:** the JS tool runs model-written code in an
  interpreter inside the app, and skills are instructions (and files) the user
  installs. Neither adds native code. On iOS, explain in the review notes that
  they don't change the app's features (guideline 2.5.2).
- **AI content reporting (Google Play):** Play's AI-generated content policy
  asks apps that generate content with AI to let users report offensive
  output. Pocket Agent has no in-app report button yet; if the review asks,
  one needs adding (for example a "Report" item on a reply's menu).
- **Sharing data with third-party AI (Apple 5.1.2(i)):** the user picks the
  provider and enters their own key before anything is sent. Say so in the
  review notes; if the review asks for explicit consent, add a confirmation
  when a cloud provider is first used.
- **Review notes:** give App Review a way to try the app without an API key:
  an on-device model (Library → download a small model such as SmolLM2), or a
  temporary key for a cloud provider.
- **Share extension:** without `IOS_SHARE_PROFILE_BASE64`, the iOS app is
  built without it, and isn't in the share sheet.
