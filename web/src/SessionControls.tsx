import {useCallback,useEffect,useState} from 'react'
import {finishInstants} from './finishTime'
import {errorMessage,request} from './api'
type Status={active:boolean;deadline:number;mode:string;playing:boolean;last:boolean}
export default function SessionControls({profile,close}:{profile:{id:string;name:string};close:()=>void}){
  const [status,setStatus]=useState<Status>()
  const parentMode=new URLSearchParams(location.search).has('parent-mode')
  const [pin,setPin]=useState('')
  const [mode,setMode]=useState('manual')
  const [minutes,setMinutes]=useState(30)
  const [finish,setFinish]=useState('')
  const [occurrence,setOccurrence]=useState(0)
  const instants=finishInstants(finish)
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  const path='profiles/'+profile.id+'/session'
  const refresh=useCallback(async()=>setStatus(await request<Status>(path)),[path])
  useEffect(()=>{refresh().catch(e=>setError(errorMessage(e)))},[refresh])
  async function change(method:string){
    setBusy(true);setError('')
    try{
      if(method==='POST')await request(path,method,{mode,minutes,...(parentMode?{pin}:{}),finish_by:mode==='finish_by'?instants[occurrence].toISOString():''})
      else await request(path,method)
      await refresh()
    }catch(e){setError(errorMessage(e))}finally{setBusy(false);setPin('')}
  }
  return <section aria-label={'Viewing session for '+profile.name}><h3>Viewing session for {profile.name}</h3>
    {status?.active ? <>
      <p>{status.last?'Ends after the current video':status.deadline?'Ends '+new Date(status.deadline).toLocaleString():'Ends when you stop it'}</p>
      <div className="actions"><button disabled={busy} onClick={()=>change('DELETE')}>End now</button>
      {!status.last&&<button disabled={busy} onClick={()=>change('PATCH')}>End after current video</button>}</div>
    </> : status && <form onSubmit={e=>{e.preventDefault();void change('POST')}}>
      {parentMode&&<label>Parent PIN<input type="password" inputMode="numeric" pattern="[0-9]{6,12}" required value={pin} onChange={e=>setPin(e.target.value)}/></label>}
      <label>Session ends<select value={mode} onChange={e=>setMode(e.target.value)}>
        <option value="manual">When I stop it</option><option value="one_video">After one video</option><option value="duration">After a duration</option><option value="finish_by">Finish by a clock time</option>
      </select></label>
      {mode==='duration'&&<label>Minutes<input type="number" min="1" max="1440" required value={minutes} onChange={e=>setMinutes(Number(e.target.value))}/></label>}
      {mode==='finish_by'&&<label>Finish by ({Intl.DateTimeFormat().resolvedOptions().timeZone})<input type="datetime-local" required value={finish} onChange={e=>{setFinish(e.target.value);setOccurrence(0)}}/></label>}
      {mode==='finish_by'&&finish&&!instants.length&&<p role="alert">That local time does not exist. Choose another minute.</p>}
      {mode==='finish_by'&&instants.length>1&&<label>Which occurrence?<select value={occurrence} onChange={e=>setOccurrence(Number(e.target.value))}>{instants.map((d,i)=><option key={d.getTime()} value={i}>{i===0?'First':'Second'} · {d.toLocaleTimeString([],{timeZoneName:'short'})}</option>)}</select></label>}
      <button disabled={busy||(mode==='finish_by'&&!instants[occurrence])}>Start session</button>
    </form>}
    {error&&<p role="alert">{error}</p>}
    <button onClick={()=>refresh().catch(e=>setError(errorMessage(e)))}>Refresh session</button>
    <button onClick={close}>Close session controls</button>
  </section>
}
