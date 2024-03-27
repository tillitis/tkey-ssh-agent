
source = ["dist/tkey-ssh-agent_darwin_all/tkey-ssh-agent"]
bundle_id = "com.tillitis.tkey-ssh-agent"

apple_id {
  username = "[email protected]"
  password = "@keychain:[email protected]"
  provider = "34722S433A"
}

sign {
  application_identity = "Developer ID Application: Tillitis AB"
}
