import {useEffect,useState} from 'react'
import {errorMessage,request} from './api'

import {Scanner,MetadataEditor} from './CatalogTools'
import Channels from './Channels'
import MediaReview,{type ReviewedMedia as Media} from './MediaReview'
export default function MediaLibrary(){
  const [media,setMedia]=useState<Media[]>([])
  const [selected,setSelected]=useState<Media>()
  const [editing,setEditing]=useState<Media>()
  const [gaps,setGaps]=useState<{series:string;season:number;from:number;to:number}[]>([])
  const [reviewing,setReviewing]=useState<Media>()
  const [path,setPath]=useState('')
  const [title,setTitle]=useState('')
  const [error,setError]=useState('')
  const [busy,setBusy]=useState(false)
  const [reload,setReload]=useState(0)
  useEffect(()=>{
    let active=true
    request<{media:Media[];gaps:typeof gaps}>('media').then(result=>{if(active){setMedia(result.media);setGaps(result.gaps)}})
      .catch(error=>{if(active)setError(errorMessage(error))})
    return ()=>{active=false}
  },[reload])
  async function add(e:React.FormEvent){
    e.preventDefault();setBusy(true);setError('')
    try{
      const item=await request<Media>('media','POST',{path,title})
      setMedia(old=>[...old.filter(m=>m.id!==item.id),item]);setPath('');setTitle('');setReload(n=>n+1)
    }catch(error){setError(errorMessage(error))}finally{setBusy(false)}
  }
  async function remove(item:Media){
    if(!window.confirm('Remove '+item.title+' from the library? The source file stays on disk.')) return
    setBusy(true);setError('')
    try{
      await request('media/'+encodeURIComponent(item.id),'DELETE')
      setMedia(previous=>previous.filter(m=>m.id!==item.id))
      if(selected?.id===item.id)setSelected(undefined)
      if(reviewing?.id===item.id)setReviewing(undefined)
      if(editing?.id===item.id)setEditing(undefined);setReload(n=>n+1)
    }catch(error){setError(errorMessage(error))}finally{setBusy(false)}
  }
  return <div>
    <h2>Media library</h2>
    <Scanner media={media} added={item=>{setMedia(old=>[...old.filter(m=>m.id!==item.id),item]);setReload(n=>n+1)}}/>
    {gaps.length>0&&<details><summary>Episode gaps</summary><ul>{gaps.map(g=><li key={g.series+g.season+'-'+g.from}>{g.series} · season {g.season} · missing {g.from}{g.to!==g.from?'–'+g.to:''}</li>)}</ul></details>}
    <form onSubmit={add}>
      <label htmlFor="media-path">Video path in your media folder</label>
      <input id="media-path" placeholder="Nature/Forest.mp4" required value={path} onChange={e=>setPath(e.target.value)}/>
      <label htmlFor="media-title">Display title (optional)</label>
      <input id="media-title" maxLength={160} value={title} onChange={e=>setTitle(e.target.value)}/>
      <button disabled={busy}>{busy?'Reading video…':'Import video'}</button>
    </form>
    {error&&<p role="alert">{error} <button onClick={()=>{setError('');setReload(n=>n+1)}}>Reload library</button></p>}
    {media.length>0 && <p aria-label="Library health">{media.filter(m=>!m.review.profiles?.length).length} awaiting approval · {media.filter(m=>!m.review.subject).length} without a subject · {media.filter(m=>!m.review.audio_description||m.review.audio_description==='unknown').length} with unknown audio description</p>}
    <ul>{media.map(item=><li key={item.id}>
      <strong>{item.title}</strong> · {Math.ceil(item.duration_seconds/60)} min{item.captions?' · captions':''}
      <button aria-label={'Catalog details for '+item.title} onClick={()=>setEditing(item)}>Details</button>
      <button aria-label={'Review '+item.title} onClick={()=>setReviewing(item)}>Review</button>
      <button aria-label={'Preview '+item.title} onClick={()=>setSelected(item)}>Preview</button>
      <button disabled={busy} aria-label={'Remove video '+item.title} onClick={()=>remove(item)}>Remove</button>
    </li>)}</ul>
    {editing&&<MetadataEditor key={editing.id} item={editing} close={()=>setEditing(undefined)} saved={item=>{setMedia(old=>old.map(m=>m.id===item.id?item:m));setEditing(undefined);setReload(n=>n+1)}}/>}
    {reviewing && <MediaReview key={reviewing.id} item={reviewing} close={()=>setReviewing(undefined)} saved={item=>{setMedia(old=>old.map(m=>m.id===item.id?item:m));setReviewing(undefined)}}/>}
    <Channels media={media}/>
    {selected && <div>
      <h3>{selected.title}</h3>
      <video key={selected.id} controls preload="metadata" src={'/api/media/'+selected.id+'/file'} aria-label={selected.title}
        onError={()=>setError('Preview unavailable. Check the source file and browser video support.')}>
        {selected.captions && <track kind="captions" label="Captions" src={'/api/media/'+selected.id+'/captions'}/>}
      </video>
      <button onClick={()=>setSelected(undefined)}>Close preview</button>
    </div>}
  </div>
}
