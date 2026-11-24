import {useCallback,useEffect,useRef,useState} from 'react'
import type {Discovery} from './DiscoverySettings'
import {ApiError,errorMessage,request} from './api'
type Item={id:string;title:string;subject:string;duration_seconds:number;captions:boolean}
type Airing={item:Item;start:number;end:number}
type Guide={id:string;name:string;now:Airing|null;next:Airing|null}
import {avatars} from './avatars'
type Catalog={avatar?:string;page:number;pages:number;discovery:Discovery;large_text:boolean;high_contrast:boolean;ending_music:boolean;channels:Guide[];live:Guide|null;tuned:boolean;items:Item[];current:Item|null;deadline:number;server_time:number;captions:boolean;navigation:string;reduced_motion:boolean}
import {facts} from './endingFacts'
const nativeTV=navigator.userAgent.includes('FamilyTV-AndroidTV')
const joinCode=location.pathname==='/watch'?location.hash.slice(1):''
if(joinCode)history.replaceState(null,'','/watch')
let joining:Promise<unknown>|undefined
export default function Viewer(){
  const [catalog,setCatalog]=useState<Catalog>()
  const [error,setError]=useState('')
  const [ended,setEnded]=useState(false)
  const preferences=useRef({large_text:false,high_contrast:false,reduced_motion:false,ending_music:false})
  const [query,setQuery]=useState('')
  const [page,setPage]=useState(0)
  const pageNumber=useRef(0)
  const [busy,setBusy]=useState(false)
  const [fact]=useState(()=>facts[Math.floor(Math.random()*facts.length)])
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
  const end=useCallback(()=>{revision.current++;setCatalog(undefined);setEnded(true);setError('')},[])
  const refresh=useCallback(async()=>{
    const n=++revision.current
    try{
      const next=await request<Catalog>('watch?'+new URLSearchParams({q:searchTerm.current,page:String(pageNumber.current)}))
      if(mounted.current&&n===revision.current){preferences.current=next;setCatalog(next);setError('')}
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
    if(!catalog?.deadline)return
    const timeout=setTimeout(end,Math.max(0,catalog.deadline-catalog.server_time))
    return()=>clearTimeout(timeout)
  },[catalog,end])
  async function action(action:'play'|'finish'|'tune',id:string){
    setBusy(true);setError('')
    try{
      const result=await request<{ended?:boolean}>('watch/'+action,'POST',{id})
      if(result.ended)end();else await refresh()
    }catch(e){if(e instanceof ApiError&&e.status===403)end();else setError(errorMessage(e))}finally{setBusy(false)}
  }
  function move(e:React.KeyboardEvent<HTMLElement>){
    if(!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(e.key)||(e.target as HTMLElement).tagName==='VIDEO'||((e.target as HTMLElement).tagName==='INPUT'&&['ArrowLeft','ArrowRight'].includes(e.key)))return
    const controls=Array.from(e.currentTarget.querySelectorAll<HTMLElement>('button:not(:disabled),a[href],input,video'))
    const index=controls.indexOf(document.activeElement as HTMLElement)
    if(controls.length){e.preventDefault();controls[Math.max(0,Math.min(controls.length-1,index+(['ArrowLeft','ArrowUp'].includes(e.key)?-1:1)))].focus()}
  }
  useEffect(()=>{
    searchTerm.current=query.trim();pageNumber.current=page
    if(!query && !catalog)return
    const id=setTimeout(()=>{if(!ended)void refresh()},200)
    return()=>clearTimeout(id)
  },[query,page])
  const live=catalog?.live?.now
  const current=catalog?.current
  return <main onFocus={e=>{if(catalog?.discovery.spoken_labels&&'speechSynthesis' in window){
    const text=(e.target as HTMLElement).getAttribute('aria-label')
    if(text){const voice=speechSynthesis.getVoices().find(v=>v.localService);if(voice){speechSynthesis.cancel();const utterance=new SpeechSynthesisUtterance(text);utterance.voice=voice;speechSynthesis.speak(utterance)}}
  }}} className={['viewer',preferences.current.large_text?'large':'',preferences.current.high_contrast?'contrast':'',preferences.current.reduced_motion?'still':''].join(' ')} onKeyDown={move}><header><strong>FamilyTV</strong>{catalog?.avatar&&<span role="img" aria-label={catalog.avatar+' profile avatar'}>{avatars[catalog.avatar]}</span>}</header>
    {ended?<section><h1>Viewing time is over</h1><h2>Did you know?</h2><p>{fact}</p>{music&&<audio controls autoPlay src={music} aria-label="Session-ending music"/>}</section>:<>
      {error&&<p role="alert">{error}</p>}
      {!catalog&&!error&&<p role="status">Connecting…</p>}
      {catalog?.deadline&&Boolean(catalog.discovery.warn_minutes)&&catalog.deadline-catalog.server_time<=(catalog.discovery.warn_minutes||0)*60000?<p>Viewing time ends soon.</p>:null}
      {catalog && (current?<section><h1>{current.title}</h1>
        {nativeTV?<a className="play-native" href="/watch/native">Play on TV</a>:<video key={current.id+':'+(live?.start||0)} autoPlay={Boolean(live)}
          onLoadedMetadata={e=>{if(live)e.currentTarget.currentTime=Math.max(0,(catalog.server_time-live.start)/1000)}} controls preload="metadata" src={'/api/watch/media/'+current.id+'/file'} onEnded={()=>live?refresh():action('finish',current.id)} onError={()=>setError('This video is unavailable.')}>
          {current.captions&&<track kind="captions" label="Captions" default={catalog.captions} src={'/api/watch/media/'+current.id+'/captions'}/>}
        </video>}
        <button disabled={busy} onClick={()=>action('finish',current.id)}>{catalog.tuned?'Leave channel':'Finish video'}</button>
      </section>:<section><h1>Choose a video</h1>
        {catalog.tuned?<><p>No approved program is on this channel right now.</p><button onClick={()=>action('finish','')}>Leave channel</button></>:<>
        {catalog.channels.length>0&&<div><h2>Live TV</h2><ul>{catalog.channels.map(c=><li key={c.id}>
          <button disabled={busy||!c.now} onClick={()=>action('tune',c.id)}>{c.name}</button>
          {c.now&&<span> · Now playing {c.now.item.title}</span>}
          {c.next&&<span> · {new Date(c.next.start).toLocaleTimeString([],{hour:'numeric',minute:'2-digit'})} {c.next.item.title}</span>}
        </li>)}</ul></div>}
        {catalog.discovery.search!=='off'&&<label>Search videos<input value={query} onChange={e=>{setQuery(e.target.value);setPage(0)}}/></label>}
        <ul>{catalog.items.filter(m=>(m.title+' '+m.subject).toLowerCase().includes(query.trim().toLowerCase())).map(m=><li key={m.id}><button aria-label={m.title} disabled={busy} onClick={()=>action('play',m.id)}>
          {catalog.navigation!=='text'&&<img loading="lazy" width="120" height="68" alt={catalog.navigation==='visual'?m.title:''} src={'/api/watch/media/'+m.id+'/picture'}/>}
          {catalog.navigation!=='visual'&&m.title}
        </button>{catalog.discovery.reading==='detail'&&m.subject&&<span> · {m.subject}</span>} · {Math.ceil(m.duration_seconds/60)} min</li>)}</ul>
        {catalog.pages>1&&<nav aria-label="Video pages"><button disabled={catalog.page===0} onClick={()=>setPage(catalog.page-1)}>Previous choices</button> {catalog.page+1} / {catalog.pages} <button disabled={catalog.page+1>=catalog.pages} onClick={()=>setPage(catalog.page+1)}>More choices</button></nav>}
        {!catalog.items.length&&<p>No videos are available for this session.</p>}</>}
      </section>)}
    </>}
  </main>
}
