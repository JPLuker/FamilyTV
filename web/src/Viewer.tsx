import Brand from './Brand'
import {TopicBadge} from './TopicBadge'
import {useCallback,useEffect,useRef,useState} from 'react'
import {WatchPlayer} from './WatchPlayer'
import type {Discovery} from './DiscoverySettings'
import {ApiError,errorMessage,request} from './api'
type Item={id:string;title:string;subject:string;duration_seconds:number;captions:boolean}
import {avatars} from './avatars'
type ShelfName='All videos'|'Favorites'|'Saved for later'|'Continue watching'
const lists:Record<ShelfName,string>={'All videos':'','Favorites':'favorites','Saved for later':'later','Continue watching':'resume'}
type Bookmark={favorite:boolean;later:boolean;position:number}
type Catalog={ending_topic:string;quiet_ending:boolean;list:string;shelf:{item:Item;bookmark:Bookmark;playable:boolean}[];resume:number;avatar?:string;page:number;pages:number;discovery:Discovery;large_text:boolean;high_contrast:boolean;ending_music:boolean;items:Item[];current:Item|null;deadline:number;server_time:number;captions:boolean;navigation:string;reduced_motion:boolean}
import {chooseFact} from './endingFacts'
const nativeTV=navigator.userAgent.includes('FamilyTV-AndroidTV')
const joinCode=location.pathname==='/watch'?location.hash.slice(1):''
if(joinCode)history.replaceState(null,'','/watch')
let joining:Promise<unknown>|undefined
export default function Viewer(){
  const [catalog,setCatalog]=useState<Catalog>()
  const [error,setError]=useState('')
  const [ended,setEnded]=useState(false)
  const preferences=useRef({ending_topic:"",quiet_ending:true,large_text:false,high_contrast:false,reduced_motion:false,ending_music:false})
  const [query,setQuery]=useState('')
  const [topic,setTopic]=useState('')
  const [searchOpen,setSearchOpen]=useState(false)
  const [shelf,setShelf]=useState<ShelfName>('All videos')
  const root=useRef<HTMLElement>(null)
  const lastTile=useRef('')
  const position=useRef<number|undefined>(undefined)
  const [page,setPage]=useState(0)
  const pageNumber=useRef(0)
  const listName=useRef('')
  const [busy,setBusy]=useState(false)
  const [fact,setFact]=useState<ReturnType<typeof chooseFact>>()
  const localDeadline=useRef(0)
  const [hardStop,setHardStop]=useState(false)
  const [showSignOff,setShowSignOff]=useState(true)
  const [music,setMusic]=useState('')
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
  const searchTerm=useRef('')
  const revision=useRef(0)
  const mounted=useRef(true)
  const end=useCallback(()=>{if(localDeadline.current && performance.now()>=localDeadline.current)setHardStop(true);revision.current++;setCatalog(undefined);setEnded(true);setError('')},[])
  const refresh=useCallback(async()=>{
    const n=++revision.current
    const sent=performance.now()
    try{
      const next=await request<Catalog>('watch?'+new URLSearchParams({q:searchTerm.current,page:String(pageNumber.current),list:listName.current}))
      if(mounted.current&&n===revision.current){next.server_time+=Math.ceil(performance.now()-sent);localDeadline.current=next.deadline?performance.now()+next.deadline-next.server_time:0;preferences.current=next;setCatalog(next);setError('')}
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
  useEffect(()=>{
    searchTerm.current=query.trim();pageNumber.current=page;listName.current=lists[shelf]
    if(!query && !catalog)return
    const id=setTimeout(()=>{if(!ended)void refresh()},200)
    return()=>clearTimeout(id)
  },[query,page,shelf])
  const current=catalog?.current
  const bookmarks=new Map(catalog?.shelf.map(row=>[row.item.id,row.bookmark]))
  const items=catalog?.list===lists[shelf]?catalog.items:[]
  const term=catalog?.discovery.search==='off'?'':query.trim().toLowerCase()
  const matching=items.filter(m=>(m.title+' '+m.subject).toLowerCase().includes(term))
  const topics=Array.from(new Set(matching.map(item=>item.subject.trim()).filter(Boolean)))
  const selectedTopic=topics.includes(topic)?topic:''
  const visible=selectedTopic?matching.filter(item=>item.subject.trim()===selectedTopic):matching
  function chooseShelf(name:ShelfName){
    setShelf(name);setPage(0);setQuery('');setTopic('');setSearchOpen(false)
  }
  async function mark(m:Item,kind:'favorite'|'later'){
    const control=document.activeElement as HTMLElement|null
    setBusy(true)
    try{await request('watch/bookmark','POST',{id:m.id,kind,enabled:!bookmarks.get(m.id)?.[kind]});await refresh()}
    catch(e){setError(errorMessage(e))}finally{
      setBusy(false)
      requestAnimationFrame(()=>{
        if(control?.isConnected)control.focus()
        else root.current?.querySelector<HTMLElement>('[data-choice],.watch-shelves [aria-pressed=true]')?.focus()
      })
    }
  }
  function goBack(){
    if(busy)return true
    const detail=root.current?.querySelector<HTMLDetailsElement>('details[open]')
    if(detail){detail.open=false;detail.querySelector('summary')?.focus();return true}
    if(current){void action('leave',current?.id||'');return true}
    if(searchOpen){setSearchOpen(false);setQuery('');setPage(0);document.getElementById('viewer-find')?.focus();return true}
    if(selectedTopic){setTopic('');document.getElementById('topic-all')?.focus();return true}
    if(shelf!=='All videos'||page>0){setShelf('All videos');setPage(0);return true}
    return false
  }
  useEffect(()=>{
    const back=(event:Event)=>{if(goBack())event.preventDefault()}
    window.addEventListener('familytv-back',back)
    return ()=>window.removeEventListener('familytv-back',back)
  })
  const nested=Boolean(current||searchOpen||selectedTopic||shelf!=='All videos'||page)
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
  },[current?.id,hasCatalog,catalog?.list])
  return <main data-input={nativeTV?"remote":undefined} ref={root} onFocus={e=>{if(catalog?.discovery.spoken_labels&&'speechSynthesis' in window){
    const text=(e.target as HTMLElement).getAttribute('aria-label')
    if(text){const voice=speechSynthesis.getVoices().find(v=>v.localService);if(voice){speechSynthesis.cancel();const utterance=new SpeechSynthesisUtterance(text);utterance.voice=voice;speechSynthesis.speak(utterance)}}
  }}} className={['viewer','child-world',ended?'night':'',ended&&preferences.current.quiet_ending?'quiet':'',preferences.current.large_text?'large':'',preferences.current.high_contrast?'contrast':'',preferences.current.reduced_motion?'still':''].join(' ')} onKeyDown={move}>{!ended&&<header><Brand/>{catalog?.avatar&&<span role="img" aria-label={catalog.avatar+' profile avatar'}>{avatars[catalog.avatar]}</span>}</header>}
    {ended?<section className="night-ending"><span className="moon" aria-hidden="true"/><h1>Viewing time is over</h1>{showSignOff&&!hardStop&&!preferences.current.quiet_ending&&<><h2>Did you know?</h2><p>{fact?.text}</p>{fact&&<small>{fact.source}</small>}{music&&<audio controls autoPlay src={music} aria-label="Session-ending music"/>}</>}</section>:<>
      {error&&<p role="alert">{error}</p>}
      {!catalog&&!error&&<p role="status">Connecting…</p>}
      {catalog?.deadline&&Boolean(catalog.discovery.warn_minutes)&&catalog.deadline-catalog.server_time<=(catalog.discovery.warn_minutes||0)*60000?<p>Viewing time ends soon.</p>:null}
      {catalog && (current?<section><h1>{current.title}</h1>
        {nativeTV?<a className="play-native" href="/watch/native">Play on TV</a>:<WatchPlayer key={current.id} id={current.id} duration={current.duration_seconds} captions={current.captions} enableCaptions={catalog.captions}
          resume={catalog.resume}
          position={seconds=>{position.current=seconds}} finished={()=>void action('finish',current.id)}/>}
        <button disabled={busy} onClick={()=>action('leave',current.id)}>Back to choices</button>
        <button disabled={busy} onClick={()=>action('finish',current.id)}>Finish video</button>
      </section>:<section className="discovery">
        <div className="discovery-heading"><h1>Choose a video</h1>
          {catalog.discovery.search!=='off'&&<button id="viewer-find" aria-expanded={searchOpen} onClick={()=>{setSearchOpen(!searchOpen);setQuery('');setPage(0)}}><span aria-hidden="true">⌕ </span>Find</button>}
        </div>
        {searchOpen&&catalog.discovery.search!=='off'&&<label className="find-box">Search videos<input autoFocus value={query} onChange={e=>{setQuery(e.target.value);setTopic('');setPage(0)}}/></label>}
        {topics.length>1&&<nav className="topic-strip" aria-label="Topics on this page">
          <button id="topic-all" aria-pressed={!selectedTopic} onClick={()=>setTopic('')}><span className="topic-art mix" aria-hidden="true">✦</span>All topics</button>
          {topics.map(name=><button key={name} aria-pressed={selectedTopic===name} onClick={()=>setTopic(selectedTopic===name?'':name)}><TopicBadge subject={name}/>{name}</button>)}
        </nav>}
        <ul className="picture-wall">{visible.map(m=><li key={m.id}>
          <button id={'choice-'+m.id} data-choice aria-label={m.title} disabled={busy} onClick={()=>action('play',m.id)}>
            {catalog.navigation!=='text'&&<span className="cover"><span className="cover-fallback">{m.title}</span><img loading="lazy" width="480" height="270" alt="" src={'/api/watch/media/'+m.id+'/picture'} onError={e=>{e.currentTarget.hidden=true}}/><span className="play-dot" aria-hidden="true">▶</span></span>}
            {catalog.navigation!=='visual'&&<strong>{m.title}</strong>}
          </button><div className="tile-foot"><span>{catalog.discovery.reading==='detail'&&m.subject&&m.subject+' · '}{Math.ceil(m.duration_seconds/60)} min</span>
          <details><summary aria-label={'Options for '+m.title}>•••</summary>
            <button disabled={busy} aria-pressed={Boolean(bookmarks.get(m.id)?.favorite)} onClick={()=>mark(m,'favorite')}>Favorite</button>
            <button disabled={busy} aria-pressed={Boolean(bookmarks.get(m.id)?.later)} onClick={()=>mark(m,'later')}>Save for later</button>
          </details></div></li>)}</ul>
        {!visible.length&&<p className="empty-shelf">{query?'No videos match your search.':shelf==='All videos'?'No videos are available for this session.':'No matching videos in this list.'}</p>}
        <nav className="watch-shelves" aria-label="Video lists">{(['All videos','Favorites','Saved for later','Continue watching'] as const).map((name,i)=><button key={name} aria-pressed={shelf===name} onClick={()=>chooseShelf(name)}><span aria-hidden="true">{['▦','♥','★','▶'][i]}</span>{name}</button>)}</nav>
        {visible.length>0&&<button className="surprise" disabled={busy} onClick={()=>action('play',visible[Math.floor(Math.random()*visible.length)].id)}>Surprise me</button>}
        {catalog.pages>1&&<nav aria-label="Video pages"><button disabled={catalog.page===0} onClick={()=>{setPage(catalog.page-1);setTopic('')}}>Previous choices</button> {catalog.page+1} / {catalog.pages} <button disabled={catalog.page+1>=catalog.pages} onClick={()=>{setPage(catalog.page+1);setTopic('')}}>More choices</button></nav>}

      </section>)}
    </>}
  </main>
}
