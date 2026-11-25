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
  await page.getByLabel('Minutes',{exact:true}).fill('25');
  await page.getByRole('heading',{name:'Viewing session for Robin'}).click();
  await page.getByRole('region',{name:'Viewing session for Robin'}).screenshot({path:'docs/guide/img/session.png'});
  const {profiles}=await (await page.request.get('/api/profiles')).json();
  const session=await (await page.request.post('/api/profiles/'+profiles[0].id+'/session',{data:{mode:'manual'}})).json();
  const child=await browser.newPage({viewport:{width:980,height:650}});
  await child.goto('http://localhost:17881/watch#'+session.code);
  await child.getByRole('heading',{name:'Choose a video'}).waitFor();
  await child.evaluate(()=>document.fonts.ready);
  await child.screenshot({path:'viewer-preview.png',animations:'disabled'});
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)})
