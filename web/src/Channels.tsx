import {useEffect,useState} from 'react'
import {errorMessage,request} from './api'
type Channel={id:string;name:string;slots:{media:string;milliseconds:number}[]}
export default function Channels({media}:{media:{id:string;title:string}[]}){
  const [channels,setChannels]=useState<Channel[]>([])
  const [name,setName]=useState('')
  const [pick,setPick]=useState('')
  const [lineup,setLineup]=useState<string[]>([])
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  useEffect(()=>{request<{channels:Channel[]}>('channels').then(x=>setChannels(x.channels)).catch(e=>setError(errorMessage(e)))},[])
  async function save(e:React.FormEvent){
    e.preventDefault();setBusy(true);setError('')
    try{const c=await request<Channel>('channels','POST',{name,media:lineup});setChannels(old=>[...old,c]);setName('');setLineup([])}
    catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  async function remove(c:Channel){
    if(!confirm('Remove '+c.name+' from the channel guide?'))return
    setBusy(true);setError('')
    try{await request('channels/'+c.id,'DELETE');setChannels(old=>old.filter(x=>x.id!==c.id))}
    catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  return <details><summary>Live TV channels</summary>
    <ul>{channels.map(c=><li key={c.id}>{c.name} · {c.slots.length} programs <button disabled={busy} aria-label={'Remove channel '+c.name} onClick={()=>remove(c)}>Remove</button></li>)}</ul>
    <form onSubmit={save}><label>Channel name<input required maxLength={80} value={name} onChange={e=>setName(e.target.value)}/></label>
      <label>Program<select value={pick} onChange={e=>setPick(e.target.value)}><option value="">Choose a video</option>{media.map(m=><option value={m.id} key={m.id}>{m.title}</option>)}</select></label>
      <button type="button" disabled={!pick||lineup.length>=200} onClick={()=>setLineup(old=>[...old,pick])}>Append program</button>
      <ol>{lineup.map((id,i)=><li key={i}>{media.find(m=>m.id===id)?.title||'Unavailable'} <button type="button" aria-label={'Remove program '+(i+1)} onClick={()=>setLineup(old=>old.filter((_,j)=>j!==i))}>Remove</button></li>)}</ol>
      <button disabled={busy||!lineup.length}>Create channel</button>
    </form>{error&&<p role="alert">{error}</p>}
  </details>
}
