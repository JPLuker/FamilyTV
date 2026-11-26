import {useEffect,useState} from 'react'
import {errorMessage,request} from './api'

export default function DevicePairing({profile,challenge,onPaired,close}:{
  profile:{id:string;name:string};challenge?:string;onPaired:()=>void;close:()=>void
}){
  const path='profiles/'+profile.id+'/devices'
  const [count,setCount]=useState(0)
  const [code,setCode]=useState('')
  const [notice,setNotice]=useState('')
  const [busy,setBusy]=useState(false)
  useEffect(()=>{
    let mounted=true
    async function refresh(){
      try{const value=await request<{count:number}>(path);if(mounted)setCount(value.count)}
      catch(e){if(mounted)setNotice(errorMessage(e))}
    }
    void refresh()
    const timer=setInterval(refresh,5000)
    return()=>{mounted=false;clearInterval(timer)}
  },[path])
  async function change(method:'POST'|'PUT'|'DELETE'){
    if(method==='DELETE'&&!window.confirm('Disconnect every paired device for '+profile.name+'?'))return
    setBusy(true);setNotice('')
    try{
      const result=await request<{code?:string}>(path,method,method==='PUT'?{challenge}:undefined)
      setCode(result.code||'')
      if(method==='DELETE'){setCount(0);setNotice('Devices disconnected.')}
      if(method==='PUT'){onPaired();setNotice('TV authorized for '+profile.name+'. Choose Open paired viewer on the TV.')}
    }catch(e){setNotice(errorMessage(e))}finally{setBusy(false)}
  }
  return <section aria-label={'Devices for '+profile.name}><h3>Devices for {profile.name}</h3>
    <p>Pair a screen once. It will wait here between viewing sessions.</p>
    {challenge&&<button disabled={busy} onClick={()=>change('PUT')}>Pair TV {challenge.slice(0,8)} with {profile.name}</button>}
    <button disabled={busy} onClick={()=>change('POST')}>{code?'Replace pairing code':'Get pairing code'}</button>
    {code&&<><p>Device pairing code <strong>{code}</strong> · valid for 10 minutes</p>
      <label>One-time viewer link<input readOnly value={location.origin+'/watch#'+code} onFocus={e=>e.target.select()}/></label>
      <p>Use this link in the child’s browser. Opening it in the parent browser signs you out.</p></>}
    <p>{count} paired {count===1?'device':'devices'}</p>
    {(count>0||code)&&<button disabled={busy} onClick={()=>change('DELETE')}>Disconnect all devices</button>}
    {notice&&<p role="status">{notice}</p>}
    <button onClick={close}>Close device setup</button>
  </section>
}
