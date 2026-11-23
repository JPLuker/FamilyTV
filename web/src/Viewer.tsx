import {useCallback,useEffect,useRef,useState} from 'react'
import {ApiError,errorMessage,request} from './api'
type Item={id:string;title:string;subject:string;duration_seconds:number;captions:boolean}
type Catalog={items:Item[];current:Item|null;deadline:number;server_time:number;captions:boolean;navigation:string;reduced_motion:boolean}
const facts=['An octopus has three hearts.','A square has four equal sides.','Earth has one natural moon.','A triangle has three sides.','An adult insect has six legs.','Water freezes at 0°C at standard atmospheric pressure.']
const joinCode=location.pathname==='/watch'?location.hash.slice(1):''
if(joinCode)history.replaceState(null,'','/watch')
let joining:Promise<unknown>|undefined
export default function Viewer(){
  const [catalog,setCatalog]=useState<Catalog>()
  const [error,setError]=useState('')
  const [ended,setEnded]=useState(false)
  const [query,setQuery]=useState('')
  const [busy,setBusy]=useState(false)
  const [fact]=useState(()=>facts[Math.floor(Math.random()*facts.length)])
  const revision=useRef(0)
  const mounted=useRef(true)
  const end=useCallback(()=>{revision.current++;setCatalog(undefined);setEnded(true);setError('')},[])
  const refresh=useCallback(async()=>{
    const n=++revision.current
    try{
      const next=await request<Catalog>('watch')
      if(mounted.current&&n===revision.current){setCatalog(next);setError('')}
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
  async function action(action:'play'|'finish',id:string){
    setBusy(true);setError('')
    try{
      const result=await request<{ended?:boolean}>('watch/'+action,'POST',{id})
      if(result.ended)end();else await refresh()
    }catch(e){if(e instanceof ApiError&&e.status===403)end();else setError(errorMessage(e))}finally{setBusy(false)}
  }
  function move(e:React.KeyboardEvent<HTMLElement>){
    if(!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(e.key)||['INPUT','VIDEO'].includes((e.target as HTMLElement).tagName))return
    const controls=Array.from(e.currentTarget.querySelectorAll<HTMLElement>('button:not(:disabled),input,video'))
    const index=controls.indexOf(document.activeElement as HTMLElement)
    if(controls.length){e.preventDefault();controls[Math.max(0,Math.min(controls.length-1,index+(['ArrowLeft','ArrowUp'].includes(e.key)?-1:1)))].focus()}
  }
  const current=catalog?.current
  return <main onKeyDown={move}><header><strong>FamilyTV</strong></header>
    {ended?<section><h1>Viewing time is over</h1><h2>Did you know?</h2><p>{fact}</p></section>:<>
      {error&&<p role="alert">{error}</p>}
      {!catalog&&!error&&<p role="status">Connecting…</p>}
      {catalog && (current?<section><h1>{current.title}</h1>
        <video key={current.id} controls preload="metadata" src={'/api/watch/media/'+current.id+'/file'} onEnded={()=>action('finish',current.id)} onError={()=>setError('This video is unavailable.')}>
          {current.captions&&<track kind="captions" label="Captions" default={catalog.captions} src={'/api/watch/media/'+current.id+'/captions'}/>}
        </video>
        <button disabled={busy} onClick={()=>action('finish',current.id)}>Finish video</button>
      </section>:<section><h1>Choose a video</h1>
        <label>Search videos<input value={query} onChange={e=>setQuery(e.target.value)}/></label>
        <ul>{catalog.items.filter(m=>(m.title+' '+m.subject).toLowerCase().includes(query.trim().toLowerCase())).map(m=><li key={m.id}><button disabled={busy} onClick={()=>action('play',m.id)}>{m.title}</button> · {Math.ceil(m.duration_seconds/60)} min</li>)}</ul>
        {!catalog.items.length&&<p>No videos are available for this session.</p>}
      </section>)}
    </>}
  </main>
}
