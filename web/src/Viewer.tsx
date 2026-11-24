import {useCallback,useEffect,useRef,useState} from 'react'
import {WatchPlayer} from './WatchPlayer'
import type {Discovery} from './DiscoverySettings'
import {ApiError,errorMessage,request} from './api'
type Item={id:string;title:string;subject:string;duration_seconds:number;captions:boolean}
type Airing={item:Item;start:number;end:number}
type Guide={id:string;name:string;now:Airing|null;next:Airing|null}
import {avatars} from './avatars'
type ShelfName='All videos'|'Favorites'|'Saved for later'|'Continue watching'
const lists:Record<ShelfName,string>={'All videos':'','Favorites':'favorites','Saved for later':'later','Continue watching':'resume'}
type Bookmark={favorite:boolean;later:boolean;position:number}
type Catalog={ending_topic:string;quiet_ending:boolean;list:string;shelf:{item:Item;bookmark:Bookmark;playable:boolean}[];resume:number;avatar?:string;page:number;pages:number;discovery:Discovery;large_text:boolean;high_contrast:boolean;ending_music:boolean;channels:Guide[];live:Guide|null;tuned:boolean;items:Item[];current:Item|null;deadline:number;server_time:number;captions:boolean;navigation:string;reduced_motion:boolean}
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
  async function action(action:'play'|'finish'|'tune'|'leave',id:string){
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
  const live=catalog?.live?.now
  const current=catalog?.current
  const bookmarks=new Map(catalog?.shelf.map(row=>[row.item.id,row.bookmark]))
  const items=catalog?.list===lists[shelf]?catalog.items:[]
  const term=catalog?.discovery.search==='off'?'':query.trim().toLowerCase()
  const visible=items.filter(m=>(m.title+' '+m.subject).toLowerCase().includes(term))
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
    if(current||catalog?.tuned){void action('leave',current?.id||'');return true}
    if(query){setQuery('');setPage(0);return true}
    if(shelf!=='All videos'||page>0){setShelf('All videos');setPage(0);return true}
    return false
  }
  useEffect(()=>{
    const back=(event:Event)=>{if(goBack())event.preventDefault()}
    window.addEventListener('familytv-back',back)
    return ()=>window.removeEventListener('familytv-back',back)
  })
  const nested=Boolean(current||catalog?.tuned||query||shelf!=='All videos'||page)
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
  }}} className={['viewer',preferences.current.large_text?'large':'',preferences.current.high_contrast?'contrast':'',preferences.current.reduced_motion?'still':''].join(' ')} onKeyDown={move}><header><strong>FamilyTV</strong>{catalog?.avatar&&<span role="img" aria-label={catalog.avatar+' profile avatar'}>{avatars[catalog.avatar]}</span>}</header>
    {ended?<section><h1>Viewing time is over</h1>{showSignOff&&!hardStop&&!preferences.current.quiet_ending&&<><h2>Did you know?</h2><p>{fact?.text}</p>{fact&&<small>{fact.source}</small>}{music&&<audio controls autoPlay src={music} aria-label="Session-ending music"/>}</>}</section>:<>
      {error&&<p role="alert">{error}</p>}
      {!catalog&&!error&&<p role="status">Connecting…</p>}
      {catalog?.deadline&&Boolean(catalog.discovery.warn_minutes)&&catalog.deadline-catalog.server_time<=(catalog.discovery.warn_minutes||0)*60000?<p>Viewing time ends soon.</p>:null}
      {catalog && (current?<section><h1>{current.title}</h1>
        {nativeTV?<a className="play-native" href="/watch/native">Play on TV</a>:<WatchPlayer key={current.id+':'+(live?.start||0)} id={current.id} duration={current.duration_seconds} captions={current.captions} enableCaptions={catalog.captions}
          resume={catalog.resume} liveOffset={live?Math.max(0,(catalog.server_time-live.start)/1000):undefined}
          position={seconds=>{position.current=seconds}} finished={()=>live?void refresh():void action('finish',current.id)}/>}
        {!catalog.tuned&&<button disabled={busy} onClick={()=>action('leave',current.id)}>Back to choices</button>}
        <button disabled={busy} onClick={()=>action('finish',current.id)}>{catalog.tuned?'Leave channel':'Finish video'}</button>
      </section>:<section><h1>Choose a video</h1>
        {catalog.tuned?<><p>No approved program is on this channel right now.</p><button onClick={()=>action('finish','')}>Leave channel</button></>:<>
        {catalog.channels.length>0&&<div><h2>Live TV</h2><ul>{catalog.channels.map(c=><li key={c.id}>
          <button disabled={busy||!c.now} onClick={()=>action('tune',c.id)}>{c.name}</button>
          {c.now&&<span> · Now playing {c.now.item.title}</span>}
          {c.next&&<span> · {new Date(c.next.start).toLocaleTimeString([],{hour:'numeric',minute:'2-digit'})} {c.next.item.title}</span>}
        </li>)}</ul></div>}
        {catalog.discovery.search!=='off'&&<label>Search videos<input value={query} onChange={e=>{setQuery(e.target.value);setPage(0)}}/></label>}
        <nav className="watch-shelves" aria-label="Video lists">{(['All videos','Favorites','Saved for later','Continue watching'] as const).map(name=><button key={name} aria-pressed={shelf===name} onClick={()=>{setShelf(name);setPage(0);setQuery('')}}>{name}</button>)}</nav>
        {visible.length>0&&<button disabled={busy} onClick={()=>action('play',visible[Math.floor(Math.random()*visible.length)].id)}>Surprise me</button>}
        <ul className="watch-grid">{visible.map(m=><li key={m.id}>
          <button id={'choice-'+m.id} data-choice aria-label={m.title} disabled={busy} onClick={()=>action('play',m.id)}>
            {catalog.navigation!=='text'&&<img loading="lazy" width="240" height="135" alt={catalog.navigation==='visual'?m.title:''} src={'/api/watch/media/'+m.id+'/picture'}/>}
            {catalog.navigation!=='visual'&&m.title}
          </button>{catalog.discovery.reading==='detail'&&m.subject&&<span>{m.subject} · </span>}{Math.ceil(m.duration_seconds/60)} min
          <details><summary aria-label={'Options for '+m.title}>Options</summary>
            <button disabled={busy} aria-pressed={Boolean(bookmarks.get(m.id)?.favorite)} onClick={()=>mark(m,'favorite')}>Favorite</button>
            <button disabled={busy} aria-pressed={Boolean(bookmarks.get(m.id)?.later)} onClick={()=>mark(m,'later')}>Save for later</button>
          </details></li>)}</ul>
        {catalog.pages>1&&<nav aria-label="Video pages"><button disabled={catalog.page===0} onClick={()=>setPage(catalog.page-1)}>Previous choices</button> {catalog.page+1} / {catalog.pages} <button disabled={catalog.page+1>=catalog.pages} onClick={()=>setPage(catalog.page+1)}>More choices</button></nav>}
        {!visible.length&&<p>{shelf==='All videos'?'No videos are available for this session.':'No matching videos in this list.'}</p>}</>}
      </section>)}
    </>}
  </main>
}
