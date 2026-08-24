import axios from 'axios'

const request = axios.create({
  baseURL: './api',
  timeout: 60000,
})

export default request
