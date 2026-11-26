import Brand from './Brand'
import {TopicBadge} from './TopicBadge'
import {useCallback,useEffect,useRef,useState} from 'react'
import {WatchPlayer} from './WatchPlayer'
import type {Discovery} from './DiscoverySettings'
import {ApiError,errorMessage,request} from './api'
type Item={id:string;title:string;subject:string;duration_seconds:number;captions:boolean}
import {avatars} from './avatars'
type Catalog={rotation:string;ending_topic:string;quiet_ending:boolean;resume:number;avatar?:string;discovery:Discovery;large_text:boolean;high_contrast:boolean;ending_music:boolean;items:Item[];current:Item|null;deadline:number;server_time:number;captions:boolean;navigation:string;reduced_motion:boolean}
import {chooseFact} from './endingFacts'
const nativePlayer=navigator.userAgent.includes('FamilyTV-AndroidTV')
const nativeTV=nativePlayer && !navigator.userAgent.includes('FamilyTV-Handheld')
const joinCode=location.pathname==='/watch'?location.hash.slice(1):''
if(joinCode)history.replaceState(null,'','/watch')
let joining:Promise<unknown>|undefined
// A stable order within a session keeps polling and remote focus predictable.
function subjectOrder(seed:string, name:string){
  let hash=2166136261
  for(const char of name.toLowerCase()+'|'+seed)hash=Math.imul(hash^char.charCodeAt(0),16777619)
  return hash>>>0
}
export default function Viewer(){
  const [catalog,setCatalog]=useState<Catalog>()
  const [error,setError]=useState('')
  const hasViewed=useRef(false)
  const [waiting,setWaiting]=useState(false)
  const [ended,setEnded]=useState(false)
  const preferences=useRef({ending_topic:"",quiet_ending:true,large_text:false,high_contrast:false,reduced_motion:false,ending_music:false})
  const strip=useRef<HTMLDivElement>(null)
  const [edges,setEdges]=useState({back:false,next:false})
  const root=useRef<HTMLElement>(null)
  const lastTile=useRef('')
  const position=useRef<number|undefined>(undefined)
  const [busy,setBusy]=useState(false)
  const [fact,setFact]=useState<ReturnType<typeof chooseFact>>()
  const localDeadline=useRef(0)
  const [hardStop,setHardStop]=useState(false)
  const [showSignOff,setShowSignOff]=useState(true)
  const [music,setMusic]=useState('')
  useEffect(()=>{
    if(!ended)return
    let cancelled=false
    let timer:ReturnType<typeof setTimeout>
    async function check(){
      try{
        const state=await request<{paired:boolean;active:boolean}>('watch/device')
        if(cancelled)return
        setWaiting(state.paired&&!hasViewed.current)
        if(state.active){setEnded(false);setWaiting(false);setShowSignOff(true);setHardStop(false);return}
        if(!state.paired)return
      }catch{ /* Retry when the household server is reachable again. */ }
      if(!cancelled)timer=setTimeout(check,2000)
    }
    void check()
    return()=>{cancelled=true;clearTimeout(timer)}
  },[ended])
  const musicEnabled=Boolean((catalog||preferences.current).ending_music)
  useEffect(()=>{
    if(!musicEnabled){setMusic('');return}
    let active=true,url=''
    const controller=new AbortController()
    fetch('/api/watch/ending-music',{credentials:'same-origin',signal:controller.signal}).then(async response=>{
      if(!active||!response.ok)return
      const blob=await response.blob()
      if(active){url=URL.createObjectURL(blob);setMusic(url)}
    }).catch(()=>{/* Ending music is optional. The visual ending remains available. */})
    return()=>{active=false;controller.abort();if(url)URL.revokeObjectURL(url)}
  },[musicEnabled])
  const revision=useRef(0)
  const mounted=useRef(true)
  const end=useCallback(()=>{if(localDeadline.current && performance.now()>=localDeadline.current)setHardStop(true);revision.current++;setCatalog(undefined);setEnded(true);setError('')},[])
  const refresh=useCallback(async()=>{
    const n=++revision.current
    const sent=performance.now()
    try{
      const next=await request<Catalog>('watch')
      if(mounted.current&&n===revision.current){next.server_time+=Math.ceil(performance.now()-sent);localDeadline.current=next.deadline?performance.now()+next.deadline-next.server_time:0;preferences.current=next;hasViewed.current=true;setCatalog(next);setError('')}
    }catch(e){
      if(mounted.current&&n===revision.current){
        if(e instanceof ApiError && e.status===403)end()
        else{setCatalog(undefined);setError('Connection lost. Waiting for FamilyTV…')}
      }
    }
  },[end])
  useEffect(()=>{
    mounted.current=true
    if(ended)return
    let cancelled=false
    let timer:ReturnType<typeof setTimeout>
    async function connect(){
      try {
        if(joinCode){joining??=request('watch/join','POST',{code:joinCode});await joining}
        if(!cancelled)await refresh()
      }catch(e){if(!cancelled){setError(errorMessage(e));setCatalog(undefined)}}
      finally{if(!cancelled)timer=setTimeout(connect,1500)}
    }
    void connect()
    function disconnected(){revision.current++;setCatalog(undefined);setError('Connection lost. Waiting for FamilyTV…')}
    window.addEventListener('offline',disconnected)
    return()=>{cancelled=true;mounted.current=false;clearTimeout(timer);window.removeEventListener('offline',disconnected)}
  },[ended,refresh])
  useEffect(()=>{
    if(!catalog?.deadline || catalog.deadline-catalog.server_time>2147483647)return
    const timeout=setTimeout(()=>{setHardStop(true);end()},Math.min(2147483647,Math.max(0,catalog.deadline-catalog.server_time)))
    return()=>clearTimeout(timeout)
  },[catalog,end])
  useEffect(()=>{
    if(!ended)return
    if(!preferences.current.quiet_ending&&!hardStop)setFact(chooseFact(preferences.current.ending_topic))
    const timer=setTimeout(()=>setShowSignOff(false),12000)
    return ()=>clearTimeout(timer)
  },[ended])
  async function action(action:'play'|'finish'|'leave',id:string){
    setBusy(true);setError('')
    try{
      if(action==='play')lastTile.current=id
      const result=await request<{ended?:boolean}>('watch/'+action,'POST',{id,position:action==='leave'?position.current:undefined})
      if(result.ended)end();else await refresh()
    }catch(e){if(e instanceof ApiError&&e.status===403)end();else setError(errorMessage(e))}finally{setBusy(false)}
  }
  function move(e:React.KeyboardEvent<HTMLElement>){
    if(e.key==='Escape'||e.key==='BrowserBack'){if(goBack())e.preventDefault();return}
    if(!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(e.key))return
    const target=e.target as HTMLElement
    if(target.tagName==='VIDEO'||(target.tagName==='INPUT'&&['ArrowLeft','ArrowRight'].includes(e.key)))return
    const card=target.closest('.picture-wall li')
    if(card){
      const sibling=e.key==='ArrowDown'&&target.hasAttribute('data-choice')?card.querySelector('summary'):
        e.key==='ArrowUp'&&target.tagName==='SUMMARY'?card.querySelector<HTMLElement>('[data-choice]'):null
      if(sibling){e.preventDefault();sibling.focus();return}
    }
    const controls=Array.from(e.currentTarget.querySelectorAll<HTMLElement>('button:not(:disabled),a[href],input,video,summary')).filter(el=>el.getClientRects().length)
    const box=target.getBoundingClientRect()
    const horizontal=e.key==='ArrowLeft'||e.key==='ArrowRight', sign=e.key==='ArrowLeft'||e.key==='ArrowUp'?-1:1
    const candidates=controls.filter(el=>el!==target).map(el=>{
      const r=el.getBoundingClientRect()
      const dx=r.left+r.width/2-box.left-box.width/2,dy=r.top+r.height/2-box.top-box.height/2
      return {el,forward:(horizontal?dx:dy)*sign,side:Math.abs(horizontal?dy:dx)}
    }).filter(c=>c.forward>2).sort((a,b)=>(a.forward+4*a.side)-(b.forward+4*b.side))
    e.preventDefault();candidates[0]?.el.focus()
  }
  const current=catalog?.current
  const visible=catalog?.items||[]
  const groups=new Map<string,{label:string;items:Item[]}>()
  for(const item of visible){
    const label=item.subject.trim(), key=label.toLowerCase()
    if(!groups.has(key))groups.set(key,{label:label||'More videos',items:[]})
    groups.get(key)!.items.push(item)
  }
  const rows=Array.from(groups.values()).sort((a,b)=>subjectOrder(catalog?.rotation||'',a.label)-subjectOrder(catalog?.rotation||'',b.label))
  const ordered=rows.flatMap(group=>group.items.map(item=>({label:group.label,item})))
  const split=Math.ceil(ordered.length/2)
  const lanes=[ordered.slice(0,split),ordered.slice(split)].filter(lane=>lane.length).map(lane=>{
    const groups:{label:string;items:Item[]}[]=[]
    for(const {label,item} of lane){
      if(groups[groups.length-1]?.label!==label)groups.push({label,items:[]})
      groups[groups.length-1].items.push(item)
    }
    return groups
  })
  const updateEdges=useCallback(()=>{
    const el=strip.current
    if(el)setEdges({back:el.scrollLeft>2,next:el.scrollLeft+el.clientWidth<el.scrollWidth-2})
  },[])
  const stripKey=ordered.map(entry=>entry.item.id).join(',')
  useEffect(()=>{
    const el=strip.current
    if(!el)return
    updateEdges()
    const observer=new ResizeObserver(updateEdges)
    observer.observe(el)
    return ()=>observer.disconnect()
  },[stripKey,current?.id,updateEdges])
  function turn(direction:number){
    const el=strip.current
    if(!el)return
    el.scrollBy({left:direction*el.clientWidth*.85,behavior:'instant'})
    updateEdges()
  }
  async function mark(m:Item){
    const control=document.activeElement as HTMLElement|null
    setBusy(true)
    try{await request('watch/bookmark','POST',{id:m.id,kind:'later',enabled:true});await refresh()}
    catch(e){setError(errorMessage(e))}finally{
      setBusy(false)
      requestAnimationFrame(()=>{
        if(control?.isConnected)control.focus()
        else root.current?.querySelector<HTMLElement>('[data-choice]')?.focus()
      })
    }
  }
  function goBack(){
    if(busy)return true
    const detail=root.current?.querySelector<HTMLDetailsElement>('details[open]')
    if(detail){detail.open=false;detail.querySelector('summary')?.focus();return true}
    if(current){void action('leave',current?.id||'');return true}
    return false
  }
  useEffect(()=>{
    const back=(event:Event)=>{if(goBack())event.preventDefault()}
    window.addEventListener('familytv-back',back)
    return ()=>window.removeEventListener('familytv-back',back)
  })
  const nested=Boolean(current)
  useEffect(()=>{
    if(nested && !history.state?.familytvViewer)history.pushState({familytvViewer:true},'',location.href)
    if(!nested && history.state?.familytvViewer)history.back()
  },[nested])
  useEffect(()=>{
    function pop(){
      if(!nested)return
      if(goBack()&&!history.state?.familytvViewer)history.pushState({familytvViewer:true},'',location.href)
    }
    window.addEventListener('popstate',pop)
    return ()=>window.removeEventListener('popstate',pop)
  })
  useEffect(()=>{
    function unfocusedBack(event:KeyboardEvent){
      if(document.activeElement===document.body && ['Escape','BrowserBack'].includes(event.key)&&goBack())event.preventDefault()
    }
    window.addEventListener('keydown',unfocusedBack)
    return ()=>window.removeEventListener('keydown',unfocusedBack)
  })
  const hasCatalog=Boolean(catalog)
  useEffect(()=>{
    position.current=undefined
    if(!hasCatalog)return
    const focus=(current?root.current?.querySelector<HTMLElement>('video,.play-native'):
      document.getElementById('choice-'+lastTile.current))||root.current?.querySelector<HTMLElement>('[data-choice]')||root.current?.querySelector<HTMLElement>('button:not(:disabled),input')
    focus?.focus({preventScroll:true})
    if(!current && lastTile.current)focus?.scrollIntoView({inline:'nearest',block:'nearest'})
  },[current?.id,hasCatalog])
  return <main data-input={nativeTV?"remote":undefined} ref={root} onFocus={e=>{if(catalog?.discovery.spoken_labels&&'speechSynthesis' in window){
    const text=(e.target as HTMLElement).getAttribute('aria-label')
    if(text){const voice=speechSynthesis.getVoices().find(v=>v.localService);if(voice){speechSynthesis.cancel();const utterance=new SpeechSynthesisUtterance(text);utterance.voice=voice;speechSynthesis.speak(utterance)}}
  }}} className={['viewer','child-world',catalog&&!current?'choosing':'',ended?'night':'',ended&&preferences.current.quiet_ending?'quiet':'',preferences.current.large_text?'large':'',preferences.current.high_contrast?'contrast':'',preferences.current.reduced_motion?'still':''].join(' ')} onKeyDown={move}>{!ended&&<header><Brand/>{catalog?.avatar&&<span role="img" aria-label={catalog.avatar+' profile avatar'}>{avatars[catalog.avatar]}</span>}</header>}
    {ended?<section className="night-ending"><span className="moon" aria-hidden="true"/><h1>{waiting?"Paired and waiting for a session":"Viewing time is over"}</h1>{!waiting&&showSignOff&&!hardStop&&!preferences.current.quiet_ending&&<><h2>Did you know?</h2><p>{fact?.text}</p>{fact&&<small>{fact.source}</small>}{music&&<audio controls autoPlay src={music} aria-label="Session-ending music"/>}</>}</section>:<>
      {error&&<p role="alert">{error}</p>}
      {!catalog&&!error&&<p role="status">Connecting…</p>}
      {catalog?.deadline&&Boolean(catalog.discovery.warn_minutes)&&catalog.deadline-catalog.server_time<=(catalog.discovery.warn_minutes||0)*60000?<p>Viewing time ends soon.</p>:null}
      {catalog && (current?<section><h1>{current.title}</h1>
        {nativePlayer?<a className="play-native" href="/watch/native">{nativeTV ? "Play on TV" : "Play video"}</a>:<WatchPlayer key={current.id} id={current.id} duration={current.duration_seconds} captions={current.captions} enableCaptions={catalog.captions}
          resume={catalog.resume}
          position={seconds=>{position.current=seconds}} finished={()=>void action('finish',current.id)}/>}
        <button disabled={busy} onClick={()=>action('leave',current.id)}>Back to choices</button>
        <button disabled={busy} onClick={()=>action('finish',current.id)}>Finish video</button>
      </section>:<section className="discovery">
        <div className="discovery-heading"><h1>Choose a video</h1>
          <nav className="quick-bar" aria-label="Viewing choices">
            {visible.length>0&&<button className="surprise" disabled={busy} onClick={()=>action('play',visible[Math.floor(Math.random()*visible.length)].id)}>Surprise me</button>}
          </nav>
        </div>
        <div className="choice-board" ref={strip} onScroll={updateEdges}>{lanes.map((lane,laneIndex)=><div className="choice-lane" key={laneIndex}>{lane.map(group=><section className="subject-row" key={group.items[0].subject.trim().toLowerCase()} aria-label={group.label}>

          <ul className="picture-wall">{group.items.map((m,index)=><li key={m.id} className={index===0?'starts-topic':undefined}>
          {index===0&&<h2 className="topic-bookend" title={group.label}>
            {catalog.navigation!=='text'&&<span className="topic-portrait">
              <TopicBadge subject={group.label}/>
              <img src={'/api/watch/media/'+m.id+'/picture'} alt="" onError={e=>{e.currentTarget.hidden=true}}/>
            </span>}
            <span className="bookend-label">{group.label}</span>
          </h2>}
          <button id={'choice-'+m.id} data-choice aria-label={m.title} disabled={busy} onClick={()=>action('play',m.id)}>
            {catalog.navigation!=='text'&&<span className="cover"><span className="cover-fallback">{m.title}</span><img loading="lazy" width="480" height="270" alt="" src={'/api/watch/media/'+m.id+'/picture'} onError={e=>{e.currentTarget.hidden=true}}/><span className="play-dot" aria-hidden="true">▶</span></span>}
            {catalog.navigation!=='visual'&&<strong>{m.title}</strong>}
          </button><div className="tile-foot"><span>{catalog.discovery.reading==='detail'&&m.subject&&m.subject+' · '}{Math.ceil(m.duration_seconds/60)} min</span>
          <details><summary aria-label={'Options for '+m.title}>•••</summary>
            <button disabled={busy} onClick={()=>mark(m)}>Save for later</button>
          </details></div></li>)}</ul></section>)}</div>)}</div>
        {(edges.back||edges.next)&&<nav className="strip-controls" aria-label="Move along the videos">
          <button aria-label="Move left" aria-disabled={!edges.back} onClick={()=>turn(-1)}>←</button>
          <button aria-label="Move right" aria-disabled={!edges.next} onClick={()=>turn(1)}>→</button>
        </nav>}
        {!visible.length&&<p className="empty-shelf">No videos are available for this session.</p>}

      </section>)}
    </>}
  </main>
}
