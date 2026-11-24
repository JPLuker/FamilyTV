import {useCallback,useEffect,useState} from 'react'
import {errorMessage,request} from './api'
type Status={active:boolean;deadline:number;mode:string;playing:boolean;last:boolean}
export default function SessionControls({profile,close}:{profile:{id:string;name:string};close:()=>void}){
  const [status,setStatus]=useState<Status>()
  const [mode,setMode]=useState('manual')
  const [minutes,setMinutes]=useState(30)
  const [finish,setFinish]=useState('')
  const [link,setLink]=useState('')
  const [notice,setNotice]=useState('')
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  const path='profiles/'+profile.id+'/session'
  const refresh=useCallback(async()=>setStatus(await request<Status>(path)),[path])
  useEffect(()=>{refresh().catch(e=>setError(errorMessage(e)))},[refresh])
  async function change(method:string){
    setBusy(true);setError('');setNotice('')
    try{
      if(method==='POST'){
        const result=await request<{code:string}>(path,method,{mode,minutes,finish_by:mode==='finish_by'?new Date(finish).toISOString():''})
        setLink(location.origin+'/watch#'+result.code)
      }else{await request(path,method);if(method==='DELETE')setLink('')}
      await refresh()
    }catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  async function pairing(method:"POST"|"DELETE"){
    setBusy(true);setError('');setNotice('')
    try{
      const result=await request<{code?:string}>(path+'/pair',method)
      setLink(result.code?location.origin+'/watch#'+result.code:'')
      setNotice(result.code?'New pairing code ready.':'Viewer access revoked. The session timing is unchanged.')
    }
    catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  return <section aria-label={'Viewing session for '+profile.name}><h3>Viewing session for {profile.name}</h3>
    {status?.active ? <>
      <p>{status.last?'Ends after the current video':status.deadline?'Ends '+new Date(status.deadline).toLocaleString():'Ends when you stop it'}</p>
      {link && <label>One-time viewer link · valid for 10 minutes<input readOnly value={link} onFocus={e=>e.target.select()}/></label>}
      {link&&<p>Opening the viewer link signs the parent out of that browser.</p>}
      <p>{link&&<>TV pairing code <strong>{link.split('#')[1]}</strong></>}</p>
      <button disabled={busy} onClick={()=>pairing('POST')}>Get a new pairing code</button>
      <button disabled={busy} onClick={()=>pairing('DELETE')}>Revoke viewer access</button>
      {notice&&<p role="status">{notice}</p>}
      <div className="actions"><button disabled={busy} onClick={()=>change('DELETE')}>End now</button>
      {!status.last&&<button disabled={busy} onClick={()=>change('PATCH')}>End after current video</button>}</div>
    </> : status && <form onSubmit={e=>{e.preventDefault();void change('POST')}}>
      <label>Session ends<select value={mode} onChange={e=>setMode(e.target.value)}>
        <option value="manual">When I stop it</option><option value="one_video">After one video</option><option value="duration">After a duration</option><option value="finish_by">Finish by a clock time</option>
      </select></label>
      {mode==='duration'&&<label>Minutes<input type="number" min="1" max="1440" required value={minutes} onChange={e=>setMinutes(Number(e.target.value))}/></label>}
      {mode==='finish_by'&&<label>Finish by ({Intl.DateTimeFormat().resolvedOptions().timeZone})<input type="datetime-local" required value={finish} onChange={e=>setFinish(e.target.value)}/></label>}
      <button disabled={busy}>Start session</button>
    </form>}
    {error&&<p role="alert">{error}</p>}
    <button onClick={()=>refresh().catch(e=>setError(errorMessage(e)))}>Refresh session</button>
    <button onClick={close}>Close session controls</button>
  </section>
}
