const { chromium }=require('playwright');
(async()=>{
 const browser=await chromium.launch({channel:'chrome',args:['--no-sandbox']});
 try {
  const page=await browser.newPage({baseURL:'http://localhost:17881',viewport:{width:1000,height:800}});
  const data={password:'local temporary screenshot account'};
  for(const [path,body] of [['setup',data],['login',data],['profiles',{name:'Robin'}]]){
   const response=await page.request.post('/api/'+path,{data:body});
   if(!response.ok())throw Error(path+' '+response.status());
  }
  await page.goto('/');
  await page.getByRole('button',{name:'Viewing session for Robin'}).click();
  await page.getByLabel('Session ends').selectOption('duration');
  await page.getByLabel('Minutes').fill('25');
  await page.getByRole('heading',{name:'Viewing session for Robin'}).click();
  await page.getByRole('region',{name:'Viewing session for Robin'}).screenshot({path:'docs/guide/img/session.png'});
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)})
