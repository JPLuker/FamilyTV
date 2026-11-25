const { chromium }=require('playwright');
(async()=>{
 const browser=await chromium.launch({channel:'chrome',args:['--no-sandbox']});
 try {
  const page=await browser.newPage({baseURL:'http://localhost:17881',viewport:{width:1000,height:800}});
  const data={password:'local temporary screenshot account'};
  for(const [path,body] of [['setup',data],['login',data],['profiles',{name:'Robin',navigation:'visual-text'}]]){
   const response=await page.request.post('/api/'+path,{data:body});
   if(!response.ok())throw Error(path+' '+response.status());
  }
  const {profiles}=await (await page.request.get('/api/profiles')).json();
  for(const [path,title,subject] of [
    ['moon.mp4','Look at the Moon','Space'],
    ['rocket.mp4','Ready for liftoff','Rockets'],
    ['astronaut.mp4','Meet an astronaut','Space'],
    ['hubble_deep_field.mp4','A sky full of galaxies','Space']]){
    const r=await page.request.post('/api/media',{data:{path,title}});if(!r.ok())throw Error(await r.text());
    const item=await r.json();
    const review=await page.request.put('/api/media/'+item.id+'/review',{data:{profiles:[profiles[0].id],subject,caption_quality:'unknown',audio_description:'unknown'}});
    if(!review.ok())throw Error(await review.text());
  }
  const session=await (await page.request.post('/api/profiles/'+profiles[0].id+'/session',{data:{mode:'manual'}})).json();
  const child=await browser.newPage({viewport:{width:1120,height:700}});
  await child.goto('http://localhost:17881/watch#'+session.code);
  await child.getByRole('heading',{name:'Choose a video'}).waitFor();
  await child.evaluate(()=>document.fonts.ready);
  await child.waitForFunction(()=>document.querySelectorAll('.cover img').length===4 && [...document.querySelectorAll('.cover img')].every(i=>i.naturalWidth>0));
  await child.screenshot({path:'viewer-preview.png',animations:'disabled'});
  await child.setViewportSize({width:390,height:844});
  await child.screenshot({path:'phone-preview.png',animations:'disabled'});
  if(await child.evaluate(()=>document.documentElement.scrollWidth>innerWidth))throw Error('horizontal overflow');
  await page.request.delete('/api/profiles/'+profiles[0].id+'/session');
  await child.getByRole('heading',{name:'Viewing time is over'}).waitFor();
  await child.screenshot({path:'ending-preview.png',animations:'disabled'});
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)})
