using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Data;
using System.Diagnostics;
using System.Drawing;
using System.Linq;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Forms;

namespace TcpRoute2Windows
{
    public partial class Form1 : Form
    {
        bool autoRestart = true; // 主要是防止用户手动关闭也会自动重启
        int errExit = 0;

        public Form1()
        {
            InitializeComponent();
        }


        private void Form1_Load(object sender, EventArgs e)
        {
            if (this.WindowState == FormWindowState.Minimized)
            {
                this.Hide(); //或者是this.Visible = false;
            }
            start();
        }

        private void button1_Click(object sender, EventArgs e)
        {
            if (checkBox1.Checked == true)
            {
                autoRestart = true;
            }
            else
            {
                autoRestart = false;
            }
            errExit = 0;
            start();
        }

        delegate void DelChangText(string buf);

        void TcpRouteExit(object sender, EventArgs e)
        {
            upStatus();
            intoText("tcproute 退出" + Environment.NewLine);

            if (autoRestart == true && checkBox1.Checked == true && errExit < 5)
            {
                errExit += 1;
                start();
            }
        }

        void start()
        {

            // ThreadStart ts = new ThreadStart(tcpRouteStart);
            // Thread th = new Thread(ts);
            //  th.Start();

            tcpRouteStart();
            upStatus();
        }

        void tcpRouteStart()
        {
            tcpRoute = new Process();
            tcpRoute.StartInfo.FileName = "TcpRoute2.exe";
            tcpRoute.StartInfo.WorkingDirectory = ".";
            tcpRoute.StartInfo.Arguments = "";
            tcpRoute.StartInfo.CreateNoWindow = true;
            // tcpRoute.StartInfo.WindowStyle = ProcessWindowStyle.Hidden;
            tcpRoute.StartInfo.UseShellExecute = false;
            tcpRoute.StartInfo.RedirectStandardOutput = true;
            tcpRoute.StartInfo.StandardOutputEncoding = Encoding.UTF8;
            tcpRoute.StartInfo.StandardErrorEncoding = Encoding.UTF8;
            tcpRoute.StartInfo.RedirectStandardError = true;

            // 注册关闭事件
            tcpRoute.EnableRaisingEvents = true;
            tcpRoute.SynchronizingObject = this;
            tcpRoute.Exited += new EventHandler(TcpRouteExit);


            tcpRoute.OutputDataReceived += new DataReceivedEventHandler(process1_ErrorDataReceived);
            tcpRoute.ErrorDataReceived += new DataReceivedEventHandler(process1_ErrorDataReceived);

            tcpRoute.Start();

            tcpRoute.BeginOutputReadLine();
            tcpRoute.BeginErrorReadLine();

            //   tcpRoute.WaitForExit();
        }

        void upStatus()
        {
            try
            {
                if (tcpRoute != null && tcpRoute.HasExited == false)
                {
                    启动ToolStripMenuItem.Enabled = false;
                    停止ToolStripMenuItem.Enabled = true;
                    button1.Enabled = false;
                    button2.Enabled = true;
                }
                else
                {
                    启动ToolStripMenuItem.Enabled = true;
                    停止ToolStripMenuItem.Enabled = false;
                    button1.Enabled = true;
                    button2.Enabled = false;
                }
            }
            catch (Exception e)
            {
                启动ToolStripMenuItem.Enabled = true;
                停止ToolStripMenuItem.Enabled = false;
                button1.Enabled = true;
                button2.Enabled = false;
            }
        }


        void intoText(string buf)
        {
            try
            {
                if (this.InvokeRequired)
                {
                    this.BeginInvoke(new DelChangText(intoText), buf);
                }
                else
                {
                    if (textBox1 != null)
                    {
                        textBox1.AppendText(buf);
                    }
                }
            }
            catch (Exception)
            {
            }
        }

        void tcpRouteClose()
        {
            try
            {
                if (tcpRoute != null && tcpRoute.HasExited == false)
                {
                    tcpRoute.CancelOutputRead();
                    tcpRoute.CancelErrorRead();

                    tcpRoute.CloseMainWindow();
                    tcpRoute.Kill();
                    // tcpRoute.Close();
                }
            }
            catch (Exception)
            {
            }
        }

        private void button2_Click(object sender, EventArgs e)
        {
            autoRestart = false;
            tcpRouteClose();
        }

        private void textBox1_TextChanged(object sender, EventArgs e)
        {
            if (textBox1.Lines.Length > 50)
            {
                textBox1.Text = textBox1.Text.Remove(0, textBox1.Lines[0].Length + Environment.NewLine.Length);
            }
            textBox1.SelectionStart = textBox1.Text.Length;
            textBox1.ScrollToCaret();
        }

        private void Form1_FormClosing(object sender, FormClosingEventArgs e)
        {
            tcpRouteClose();
        }

        private void process1_ErrorDataReceived(object sender, DataReceivedEventArgs e)
        {
            if (!String.IsNullOrEmpty(e.Data))
            {
                intoText(e.Data + Environment.NewLine);
            }
        }

        private void notifyIcon1_DoubleClick(object sender, EventArgs e)
        {
            if (this.WindowState == FormWindowState.Minimized)
            {
                this.Show();
                this.WindowState = FormWindowState.Normal;
            }
            else
            {
                this.WindowState = FormWindowState.Minimized;
            }
        }

        private void Form1_SizeChanged(object sender, EventArgs e)
        {
            if (this.WindowState == FormWindowState.Minimized)
            {
                this.Hide(); //或者是this.Visible = false;
            }
        }

        private void 启动ToolStripMenuItem_Click(object sender, EventArgs e)
        {
            if (checkBox1.Checked == true)
            {
                autoRestart = true;
            }
            else
            {
                autoRestart = false;
            }
            errExit = 0;
            start();
        }

        private void 停止ToolStripMenuItem_Click(object sender, EventArgs e)
        {
            autoRestart = false;
            tcpRouteClose();
        }

        private void 退出ToolStripMenuItem_Click(object sender, EventArgs e)
        {
            this.Close();
        }

        private void 重新启动ToolStripMenuItem_Click(object sender, EventArgs e)
        {
            autoRestart = false;
            tcpRouteClose();
            errExit = 0;
            start();
        }

        private void button3_Click(object sender, EventArgs e)
        {
            autoRestart = false;
            tcpRouteClose();
            errExit = 0;
            start();
        }
    }
}
