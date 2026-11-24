import {defineConfig} from '@playwright/test'

export default defineConfig({
  testDir:'./tests',
  workers:1,
  use:{channel:process.env.FAMILYTV_TEST_BROWSER,baseURL:'http://127.0.0.1:17881', viewport:{width:390,height:844}},
  webServer:{
    command:'bash ../scripts/browser-test-server.sh',
    url:'http://127.0.0.1:17881/api/health',
    reuseExistingServer:false,
  },
})
